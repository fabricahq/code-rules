#!/bin/bash
# Add the demo library as a source with rule selectors, then sync.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$APP"

# A selector is a rule ID, or a prefix that ends in /*. These two are rejected before anything is written.
expect_failure code-rules project add source acme --repository git@demo.invalid:acme-library.git --rules 'techs/go*' --non-interactive
expect_failure code-rules project add source acme --repository git@demo.invalid:acme-library.git --rules techs/go --non-interactive

# Every practice rule group, plus one Go rule.
show code-rules project add source acme --repository git@demo.invalid:acme-library.git \
  --rules 'practices/*' --rules techs/go/wrap-errors --non-interactive
show cat .code-rules/config.yaml
show code-rules project sync
echo
echo "Rules the project now has:"
find .code-rules/generated/rules/acme -name '*.md' | sort
