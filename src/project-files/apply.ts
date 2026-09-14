/** @fileoverview Serializes project writers and applies staged directories with recoverable rollback. */

import { mkdir, rename, rm, lstat, open } from 'node:fs/promises';
import { join } from 'node:path';
import { hostname } from 'node:os';
import { randomUUID } from 'node:crypto';
import { object, json } from '../formats/validation';
import {
  ProjectError,
  missing,
  readBytes,
  readTree,
  treeDigest,
  textFile,
  writeTree,
} from './files';

const transaction = '.code-rules-transaction';
const lockName = '.code-rules-lock';
const cleanupName = '.code-rules-cleanup';
const transactionLimits = {
  entries: 120032,
  bytes: 1025 * 1024 * 1024,
  depth: 65,
};
const targets = ['vendor', 'generated'] as const;
type Target = (typeof targets)[number];
type Entry = { name: Target; before: string; after: string; existed: boolean };

async function exists(path: string): Promise<boolean> {
  try {
    await lstat(path);
    return true;
  } catch (error) {
    if (missing(error)) return false;
    throw error;
  }
}

async function durableJson(path: string, data: unknown): Promise<void> {
  const handle = await open(path, 'wx', 0o600);
  try {
    await handle.writeFile(JSON.stringify(data) + '\n');
    await handle.sync();
  } finally {
    await handle.close();
  }
}

/** Read a transaction only after rejecting links and special files throughout its tree. */
async function journal(root: string): Promise<Entry[] | undefined> {
  const tree = await readTree(join(root, transaction), transactionLimits);
  if (tree === null) return undefined;
  const bytes = tree.files.get('journal.json');
  if (!bytes) {
    if (
      tree.directories.some((path) => path.startsWith('old-')) ||
      tree.files.has('committed.json')
    )
      throw new ProjectError(
        'recovery-required',
        'Missing journal with retained output; preserve transaction files for manual recovery.',
      );
    // No live target is changed until journal.json has been fully written.
    await rm(join(root, transaction), { recursive: true });
    return undefined;
  }
  const record = object(
    json(textFile(bytes, 'journal.json'), 'journal.json'),
    'journal.json',
  );
  if (record.formatVersion !== 1 || !Array.isArray(record.entries))
    throw new ProjectError(
      'recovery-required',
      'Invalid transaction journal; preserve it for manual recovery.',
    );
  const raw: unknown[] = record.entries;
  const entries = raw.map((value): Entry => {
    const item = object(value, 'journal entry');
    if (
      (item.name !== 'vendor' && item.name !== 'generated') ||
      typeof item.existed !== 'boolean' ||
      typeof item.before !== 'string' ||
      !/^[a-f0-9]{64}$/u.test(item.before) ||
      typeof item.after !== 'string' ||
      !/^[a-f0-9]{64}$/u.test(item.after)
    )
      throw new ProjectError(
        'recovery-required',
        'Invalid transaction entry; preserve it for manual recovery.',
      );
    return {
      name: item.name,
      existed: item.existed,
      before: item.before,
      after: item.after,
    };
  });
  if (
    entries.length === 0 ||
    entries.length > 2 ||
    new Set(entries.map((e) => e.name)).size !== entries.length
  )
    throw new ProjectError('recovery-required', 'Invalid transaction targets.');
  return entries;
}

/** Restore interrupted replacements, or finish cleanup after a durable commit marker. Requires the writer lock. */
export async function recover(root: string): Promise<void> {
  const garbage = join(root, cleanupName);
  if ((await readTree(garbage, transactionLimits)) !== null)
    await rm(garbage, { recursive: true });
  const entries = await journal(root);
  if (!entries) return;
  const tx = join(root, transaction);
  const committed = await exists(join(tx, 'committed.json'));
  if (committed) {
    if (
      textFile(await readBytes(join(tx, 'committed.json')), 'commit marker') !==
      'true\n'
    )
      throw new ProjectError(
        'recovery-required',
        'Invalid commit marker; preserve transaction files for recovery.',
      );
  }
  // Validate every restoration before changing any target. Never erase a user's edits after an interruption.
  for (const entry of entries) {
    const current = treeDigest(await readTree(join(root, entry.name)));
    const backup = await readTree(join(tx, `old-${entry.name}`));
    if (committed) continue;
    if (backup !== null && treeDigest(backup) !== entry.before)
      throw new ProjectError(
        'recovery-required',
        `Backup for ${entry.name} changed; manual recovery required.`,
      );
    const absent = treeDigest(null);
    if (
      backup !== null
        ? current !== entry.after && current !== absent
        : current !== entry.before &&
          !(entry.existed === false && current === entry.after)
    )
      throw new ProjectError(
        'recovery-required',
        `${entry.name} changed after interruption; manual recovery required.`,
      );
  }
  if (!committed)
    for (const entry of [...entries].reverse()) {
      const backup = join(tx, `old-${entry.name}`);
      if (await exists(backup)) {
        await rm(join(root, entry.name), { recursive: true, force: true });
        await rename(backup, join(root, entry.name));
      } else if (!entry.existed)
        await rm(join(root, entry.name), { recursive: true, force: true });
    }
  if (committed) await finishCommitted(root);
  else await rm(tx, { recursive: true });
}

/** Atomically retire committed recovery state before deleting files, so interrupted cleanup can never trigger rollback. */
async function finishCommitted(root: string): Promise<void> {
  const garbage = join(root, cleanupName);
  await rename(join(root, transaction), garbage);
  await rm(garbage, { recursive: true });
}

/** Refuse read-only operations while a writer or interrupted transaction can expose mixed output. */
export async function requireIdle(root: string): Promise<void> {
  if (
    (await exists(join(root, lockName))) ||
    (await exists(join(root, transaction))) ||
    (await exists(join(root, cleanupName)))
  )
    throw new ProjectError(
      'busy',
      'A write is active or recovery is pending. Run sync or build after the writer exits.',
    );
}

/** Hold an exclusive writer lock. Recover a dead same-host owner; ambiguous ownership fails closed. */
export async function withWriter<T>(
  root: string,
  operation: () => Promise<T>,
): Promise<T> {
  const lock = join(root, lockName);
  const token = randomUUID();
  try {
    await mkdir(lock);
  } catch (error) {
    if (!(error instanceof Error && 'code' in error && error.code === 'EEXIST'))
      throw error;
    await readTree(lock);
    let owner;
    try {
      owner = object(
        json(
          textFile(await readBytes(join(lock, 'owner.json')), 'lock'),
          'lock',
        ),
        'lock',
      );
    } catch {
      throw new ProjectError(
        'busy',
        'Lock ownership is incomplete; verify no writer is running before removing .code-rules-lock.',
      );
    }
    if (
      owner.host !== hostname() ||
      !Number.isSafeInteger(owner.pid) ||
      typeof owner.pid !== 'number' ||
      owner.pid <= 0
    )
      throw new ProjectError(
        'busy',
        'Project lock belongs to another host or has invalid ownership.',
      );
    try {
      process.kill(owner.pid, 0);
      throw new ProjectError('busy', 'Another writer is using this project.');
    } catch (failure) {
      if (!(
        failure instanceof Error &&
        'code' in failure &&
        failure.code === 'ESRCH'
      ))
        throw failure;
    }
    // A recovery claim serializes contenders before reclaiming a dead owner. It is never stolen automatically.
    try {
      await mkdir(join(lock, 'recovery-claim'));
    } catch {
      throw new ProjectError(
        'busy',
        'Another process is recovering this project lock.',
      );
    }
    const claimedOwner = object(
      json(textFile(await readBytes(join(lock, 'owner.json')), 'lock'), 'lock'),
      'lock',
    );
    if (claimedOwner.token !== owner.token || claimedOwner.pid !== owner.pid) {
      await rm(join(lock, 'recovery-claim'), { recursive: true });
      throw new ProjectError('busy', 'Lock ownership changed during recovery.');
    }
    await rm(lock, { recursive: true });
    return withWriter(root, operation);
  }
  try {
    await durableJson(join(lock, 'owner.json'), {
      host: hostname(),
      pid: process.pid,
      token,
    });
    await recover(root);
    return await operation();
  } finally {
    // This lock directory is owned by this invocation; external deletion/replacement is unsupported.
    await rm(lock, { recursive: true, force: true });
  }
}

/** Stage complete targets, validate unchanged input, then replace and roll back on failure. */
export async function applyChanges(
  root: string,
  output: Partial<Record<Target, ReadonlyMap<string, Uint8Array>>>,
  assertUnchanged: () => Promise<void>,
): Promise<void> {
  const tx = join(root, transaction);
  await mkdir(tx);
  let prepared = false;
  try {
    const entries: Entry[] = [];
    for (const name of targets) {
      const files = output[name];
      if (!files) continue;
      const before = await readTree(join(root, name));
      await writeTree(join(tx, `new-${name}`), files);
      entries.push({
        name,
        before: treeDigest(before),
        existed: before !== null,
        after: treeDigest(await readTree(join(tx, `new-${name}`))),
      });
    }
    await assertUnchanged();
    await durableJson(join(tx, 'journal.json'), { formatVersion: 1, entries });
    prepared = true;
    for (const entry of entries) {
      if (entry.existed)
        await rename(join(root, entry.name), join(tx, `old-${entry.name}`));
      await rename(join(tx, `new-${entry.name}`), join(root, entry.name));
    }
    await durableJson(join(tx, 'commit-ready.json'), true);
    await rename(join(tx, 'commit-ready.json'), join(tx, 'committed.json'));
  } catch (error) {
    try {
      if (prepared) await recover(root);
      else await rm(tx, { recursive: true, force: true });
    } catch (recoveryError) {
      throw new ProjectError(
        'recovery-required',
        'Apply failed and recovery could not finish. Preserve .code-rules-transaction and retry after resolving the filesystem problem.',
        { cause: new AggregateError([error, recoveryError]) },
      );
    }
    throw error;
  }
  // Once committed, cleanup failure cannot turn success into a reported rollback. Next write retries cleanup.
  try {
    await finishCommitted(root);
  } catch {
    /* Durable commit marker makes the next recovery cleanup-only. */
  }
}
