#!/bin/bash
# Author the library group and rule: Creates a group and a complete original rule in the library. These files could later be committed to a publisher repository.
set -euo pipefail
cd "${CODE_RULES_DEMO_ROOT:?Run Prepare first.}/library"
"$CODE_RULES_DEMO_BINARY" library add group techs/go --name Go --description 'Shared Go conventions.' --when-to-read 'When writing Go.'
"$CODE_RULES_DEMO_BINARY" library add rule techs/go/return-errors --title 'Return errors to the caller' --impact HIGH --impact-description 'Keep failures visible.' --when-to-read 'When calling fallible operations.' --body-file "$CODE_RULES_DEMO_ROOT/project/body.md"
