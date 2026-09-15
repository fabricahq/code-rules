/** @fileoverview Creates a temporary imported workspace for inspecting original bytes alongside Builds output. */

import { mkdtemp, mkdir, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import {
  addLibrary,
  createImportFixture,
  fixtureSource,
  removeImportFixture,
  runImport,
} from '../fixtures/import-fixture';
import { object } from '../../src/formats/validation';

const fixture = await createImportFixture();
try {
  await addLibrary(fixture, 'one');
  await addLibrary(fixture, 'two');
  const configuration = {
    schemaVersion: 1,
    sources: {
      fabrica: fixtureSource('one'),
      acme: {
        ...fixtureSource('two'),
        repository: 'https://gitlab.com/fixture/nested/two.git',
      },
    },
  };
  const result = object(
    runImport(
      fixture,
      configuration,
      { build: true },
      {
        ...fixture.env,
        GIT_CONFIG_COUNT: '3',
        GIT_CONFIG_KEY_2: `url.file://${fixture.root}/.insteadOf`,
        GIT_CONFIG_VALUE_2: 'https://gitlab.com/fixture/nested/',
      },
    ),
    'manual import',
  );
  if ('error' in result) throw new Error(String(result['message']));
  const workspace = await mkdtemp(
    join(tmpdir(), 'code-rules-imports-example-'),
  );
  await writeFile(
    join(workspace, 'config.json'),
    JSON.stringify(configuration, null, 2),
  );
  for (const [source, rawFiles] of Object.entries(
    object(result['files'], 'files'),
  )) {
    for (const [path, encoded] of Object.entries(object(rawFiles, source))) {
      if (typeof encoded !== 'string')
        throw new Error('Expected encoded file bytes.');
      const target = join(workspace, 'vendor', source, path);
      await mkdir(dirname(target), { recursive: true });
      await writeFile(target, Buffer.from(encoded, 'base64'));
    }
  }
  for (const [path, content] of Object.entries(
    object(result['generated'], 'generated'),
  )) {
    if (typeof content !== 'string')
      throw new Error('Expected generated Markdown.');
    const target = join(workspace, 'generated', path);
    await mkdir(dirname(target), { recursive: true });
    await writeFile(target, content);
  }
  process.stdout.write(
    `Inspect the imported workspace: ${workspace}\nStart at: ${join(workspace, 'generated/RULES.md')}\nThis demonstration does not implement Workspace metadata or integrity checks.\n`,
  );
} finally {
  await removeImportFixture(fixture);
}
