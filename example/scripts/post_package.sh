#!/bin/sh
# fpack post_package hook: runs after each target's artifacts exist.
# FPACK_TARGET, FPACK_ARTIFACT (first file) and FPACK_ARTIFACTS are set.
echo "$FPACK_TARGET -> $(basename "$FPACK_ARTIFACT") ($FPACK_VERSION+$FPACK_BUILD_NUMBER)" >> build/fpack-hooks.log
