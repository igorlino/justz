# Bare `just tool` (and `jz tool`) lists the group: without this, it would run
# the first recipe.
_default:
    @justz help tool

[doc('Remove build output and caches')]
[group('tools')]
clean:
    @echo clean
