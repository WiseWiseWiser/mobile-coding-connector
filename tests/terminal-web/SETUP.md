# Scenario

Each leaf owns an isolated manager, real foreground shell command and httptest
server. No global environment or live server sessions are changed. A web
attachment uses `attach_mode=web`; explicit deletion uses REST DELETE.
