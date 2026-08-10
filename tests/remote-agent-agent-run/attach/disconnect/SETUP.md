# Scenario

**Feature**: unclean attach disconnect surfaces a clear error

```
FakeAttach abnormal close -> client AttachErr non-empty / non-zero
```

## Preconditions

- Live session seeded; TTYMode=`unclean` closes WS with non-normal code.

## Steps

1. Leaf configures unclean FakeAttach hold then close.
