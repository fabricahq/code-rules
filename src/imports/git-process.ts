/** @fileoverview Runs bounded noninteractive Git processes and discards potentially secret diagnostics. */

import { spawn } from 'node:child_process';
import { ImportError, requireActive } from './errors';

/** Captured stdout and exit status; stderr is drained but never returned because rewritten URLs may contain credentials. */
export type GitResult = { readonly output: Buffer; readonly status: number };

/** Preserve user credentials and transport routing, but never inherit another checkout's object database or index. */
function gitEnvironment(): NodeJS.ProcessEnv {
  const environment = { ...process.env };
  for (const name of [
    'GIT_DIR',
    'GIT_COMMON_DIR',
    'GIT_WORK_TREE',
    'GIT_INDEX_FILE',
    'GIT_OBJECT_DIRECTORY',
    'GIT_ALTERNATE_OBJECT_DIRECTORIES',
    'GIT_SHALLOW_FILE',
    'GIT_NAMESPACE',
  ])
    delete environment[name];
  return {
    ...environment,
    GIT_TERMINAL_PROMPT: '0',
    GCM_INTERACTIVE: 'Never',
    GIT_NO_REPLACE_OBJECTS: '1',
  };
}

/** Run Git with argument boundaries, bounded output, and process-group cleanup on cancellation. */
export async function gitProcess(
  cwd: string,
  args: ReadonlyArray<string>,
  signal: AbortSignal,
  maxBytes: number,
  input: string = '',
): Promise<GitResult> {
  requireActive(signal);
  return new Promise((resolve, reject) => {
    const child = spawn(
      'git',
      [
        '-c',
        'core.hooksPath=/dev/null',
        '-c',
        'protocol.allow=never',
        '-c',
        'protocol.https.allow=always',
        '-c',
        'protocol.ssh.allow=always',
        '-c',
        'protocol.ext.allow=never',
        ...args,
      ],
      {
        cwd,
        detached: true,
        env: gitEnvironment(),
        stdio: ['pipe', 'pipe', 'pipe'],
      },
    );
    const chunks: Array<Buffer> = [];
    let bytes = 0;
    let failure: ImportError | null = null;
    /** Stop the complete Git process group, including credential helpers holding its pipes open. */
    function stop(error: ImportError): void {
      failure ??= error;
      if (child.pid !== undefined) {
        try {
          process.kill(-child.pid, 'SIGKILL');
        } catch {
          child.kill('SIGKILL');
        }
      }
    }
    /** Translate caller cancellation without exposing the caller's arbitrary reason text. */
    function abort(): void {
      try {
        requireActive(signal);
      } catch (error) {
        if (error instanceof ImportError) stop(error);
      }
    }
    signal.addEventListener('abort', abort, { once: true });
    if (signal.aborted) abort();
    child.stdout.on('data', (chunk: Buffer) => {
      bytes += chunk.length;
      if (bytes > maxBytes)
        stop(
          new ImportError(
            'limit-exceeded',
            'Git output exceeds the import limit.',
          ),
        );
      else chunks.push(chunk);
    });
    child.stderr.on('data', (chunk: Buffer) => {
      bytes += chunk.length;
      if (bytes > maxBytes)
        stop(
          new ImportError(
            'limit-exceeded',
            'Git output exceeds the import limit.',
          ),
        );
    });
    child.stdin.on('error', () => {
      /* Exit status owns a failed Git reader, including early stdin closure. */
    });
    child.on('error', () => {
      failure ??= new ImportError(
        'git-unavailable',
        'Cannot start Git; install Git 2.30 or later and check PATH.',
      );
    });
    child.on('close', (status) => {
      signal.removeEventListener('abort', abort);
      if (failure !== null) reject(failure);
      else resolve({ output: Buffer.concat(chunks), status: status ?? 1 });
    });
    child.stdin.end(input);
  });
}
