// Run in an isolated ai-critic browser tab via browser-agent session run.
// Start with at most five test terminals. Re-run after refreshing the page to
// verify URL restoration and recovery. Results survive reload in sessionStorage.
setTimeout(async () => {
    const key = 'terminal-recovery-verification';
    const previous = JSON.parse(sessionStorage.getItem(key) || 'null');
    const report = previous || { checks: [] };
    const save = () => sessionStorage.setItem(key, JSON.stringify(report));
    const wait = (predicate, message) => new Promise((resolve, reject) => {
        const deadline = Date.now() + 15000;
        const check = () => {
            if (predicate()) resolve();
            else if (Date.now() > deadline) reject(Error(message));
            else setTimeout(check, 50);
        };
        check();
    });
    const activeText = () => document.querySelector('.terminal-instance.active')?.innerText || '';
    const selected = () => document.querySelector('.terminal-tab-item.active .terminal-tab-name')?.textContent;
    const connected = () => document.querySelector('.terminal-connection-status')?.innerText.includes('Connected');
    const check = (condition, label) => {
        if (!condition) throw Error(label);
        report.checks.push(label); save();
    };
    try {
        if (!previous) {
            while (document.querySelectorAll('.terminal-tab-item').length < 5) {
                const before = document.querySelectorAll('.terminal-tab-item').length;
                document.querySelector('.terminal-tab-add').click();
                await wait(() => document.querySelectorAll('.terminal-tab-item').length === before + 1 && connected(), 'create tab');
            }
            const fifth = [...document.querySelectorAll('.terminal-tab-item')].find(el => el.innerText.includes('Terminal 5'));
            fifth.click();
            await wait(() => selected() === 'Terminal 5' && new URL(location.href).searchParams.get('terminal') === 'session-5' && connected(), 'select Terminal 5');
            const command = `python3 -u -c 'import os,http.server; s=http.server.HTTPServer(("127.0.0.1",0),http.server.SimpleHTTPRequestHandler); print("VERIFY_SERVER_PID", os.getpid(), "PORT", s.server_port, flush=True); s.serve_forever()'\n`;
            const resp = await fetch('/api/terminal/sessions/session-5/input', { method: 'POST', body: command });
            check(resp.ok, 'foreground command accepted');
            await wait(() => /VERIFY_SERVER_PID \d+ PORT \d+/.test(activeText()), 'foreground server startup');
            const match = activeText().match(/VERIFY_SERVER_PID (\d+) PORT (\d+)/);
            report.pid = match[1]; report.port = match[2]; report.url = location.href;
            report.phase = 'refresh-required';
            check(connected() && !activeText().includes('[Terminal exited]'), 'foreground server blocks normally without reporting exit');
            save();
        } else if (previous.phase === 'refresh-required') {
            await wait(() => selected() === 'Terminal 5' && connected(), 'refresh selection restoration');
            check(new URL(location.href).searchParams.get('terminal') === 'session-5', 'refresh keeps selected backend ID in URL');
            check(new URL(location.href).searchParams.get('foo') === new URL(report.url).searchParams.get('foo'), 'unrelated query parameters survive selection and refresh');
            check(activeText().includes(`VERIFY_SERVER_PID ${report.pid} PORT ${report.port}`), 'refresh replays the same foreground server');
            check(!activeText().includes('attach_role'), 'protocol controls are not rendered as JSON');
            const sessions = await (await fetch('/api/terminal/sessions')).json();
            check(sessions.total === 5 && sessions.sessions.every(s => s.status === 'running'), 'refresh does not create orphan sessions or reap shells');
            if (!window.__terminalSockets) {
                window.__terminalSockets = [];
                const NativeWebSocket = window.WebSocket;
                window.WebSocket = class extends NativeWebSocket {
                    constructor(...args) {
                        super(...args);
                        window.__terminalSockets.push(this);
                        this.addEventListener('message', event => {
                            if (typeof event.data !== 'string') return;
                            try {
                                const msg = JSON.parse(event.data);
                                if (msg.type === 'session_state' && msg.status === 'running') this.__terminalReady = true;
                            } catch { /* PTY text, not a control frame. */ }
                        });
                    }
                };
                window.dispatchEvent(new Event('pageshow'));
                await wait(() => connected() && window.__terminalSockets.length >= 5 && window.__terminalSockets.some(s => s.url.includes('session_id=session-5') && s.__terminalReady), 'instrumented resume');
            }
            const sockets = [...window.__terminalSockets];
            check(Array.isArray(sockets), 'socket instrumentation installed for recovery scenario');
            sockets.filter(s => s.url.includes('session_id=session-5')).forEach(s => s.close());
            await wait(() => connected() && window.__terminalSockets.some(s => s.url.includes('session_id=session-5') && s.readyState === 1 && s.__terminalReady && !sockets.includes(s)), 'automatic reconnect');
            check(activeText().includes(`VERIFY_SERVER_PID ${report.pid} PORT ${report.port}`), 'socket loss automatically recovers same running foreground server');
            const count = window.__terminalSockets.length;
            window.dispatchEvent(new Event('pageshow'));
            await wait(() => connected() && window.__terminalSockets.slice(count).some(s => s.url.includes('session_id=session-5') && s.__terminalReady), 'resume stale OPEN recovery');
            check(!activeText().includes('[Terminal exited]'), 'page resume replaces OPEN socket without killing the PTY');
            // End another session while Terminal 5 remains selected.
            await fetch('/api/terminal/sessions/session-1/input', { method: 'POST', body: 'exit\n' });
            await wait(() => document.querySelector('.terminal-instance[data-terminal-id="term-session-1"]')?.textContent.includes('[Terminal exited]'), 'background actual exit');
            await wait(() => connected() && selected() === 'Terminal 5', 'active status after background exit');
            check(connected() && selected() === 'Terminal 5', 'background session exit cannot overwrite active connection status');
            report.phase = 'complete'; save();
        }
    } catch (error) {
        report.error = String(error); report.phase = 'failed'; save();
    }
}, 0);
'Verification started; inspect sessionStorage terminal-recovery-verification';
