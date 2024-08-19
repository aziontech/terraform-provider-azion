#!/bin/bash

# install tflint: 
# linux: curl -s https://raw.githubusercontent.com/terraform-linters/tflint/master/install_linux.sh | bash
# macos: brew install tflint
exit_code=0
for DIR in $(find ./examples -type f -name '*.tf' -exec dirname {} \; | sort -u); do
  pushd "$DIR"
  tflint \
    --enable-rule=terraform_comment_syntax \
    --enable-rule=terraform_deprecated_index \
    --enable-rule=terraform_deprecated_interpolation \
    --disable-rule=terraform_unused_declarations \
    --disable-rule=terraform_required_version \
    --disable-rule=terraform_required_providers \
    --disable-rule=terraform_typed_variables \
    || exit_code=1
      popd
    done
