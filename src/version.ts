/** @fileoverview Supplies the package version consistently to CLI output and generated provenance. */

import metadata from '../package.json';

/** Version embedded when packaging the executable, or read from the checkout during development. */
export const toolVersion: string = metadata.version;
