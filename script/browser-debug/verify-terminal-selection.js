// Run only on the isolated five-terminal scene AFTER verify-terminal-recovery.js.
// Exercises URL navigation and explicit deletion; deliberately closes Terminal 5.
setTimeout(async () => {
    const report = { checks: [] };
    const save = () => sessionStorage.setItem('terminal-selection-verification', JSON.stringify(report));
    const wait = (predicate, label) => new Promise((resolve, reject) => {
        const deadline = Date.now() + 10000;
        const poll = () => {
            if (predicate()) resolve();
            else if (Date.now() > deadline) reject(Error(label));
            else setTimeout(poll, 30);
        };
        poll();
    });
    const active = () => document.querySelector('.terminal-tab-item.active .terminal-tab-name')?.textContent;
    const state = () => document.querySelector('.terminal-connection-status')?.innerText;
    const tab = name => [...document.querySelectorAll('.terminal-tab-item')].find(el => el.querySelector('.terminal-tab-name')?.textContent === name);
    const selected = () => new URL(location.href).searchParams.get('terminal');
    const check = (value, label) => { if (!value) throw Error(label); report.checks.push(label); save(); };
    try {
        tab('Terminal 1').click();
        await wait(() => active() === 'Terminal 1' && state().includes('Exited'), 'real exited session selected');
        check(selected() === 'session-1', 'confirmed exited session is labeled Exited, not Connected');
        tab('Terminal 5').click();
        await wait(() => selected() === 'session-5' && state().includes('Connected'), 'return to running session');
        history.pushState({}, '', '/terminal?foo=preserved&terminal=session-2');
        dispatchEvent(new PopStateEvent('popstate'));
        await wait(() => active() === 'Terminal 2', 'external URL selection');
        check(selected() === 'session-2', 'external URL navigation selects requested terminal');
        history.back();
        await wait(() => active() === 'Terminal 5' && selected() === 'session-5', 'Back selects previous terminal');
        check(true, 'browser Back restores prior terminal selection');
        tab('Terminal 5').querySelector('.terminal-tab-close').click();
        await wait(() => active() === 'Terminal 4' && selected() === 'session-4', 'close active tab');
        const afterClose = await (await fetch('/api/terminal/sessions')).json();
        check(afterClose.total === 4 && !afterClose.sessions.some(s => s.id === 'session-5'), 'closing selected tab explicitly deletes session and selects adjacent tab');
        tab('Terminal 2').querySelector('.terminal-tab-close').click();
        await wait(() => !tab('Terminal 2'), 'close inactive tab');
        check(active() === 'Terminal 4' && selected() === 'session-4', 'closing inactive tab preserves selected terminal');
        history.pushState({}, '', '/terminal?foo=preserved&terminal=missing-session');
        dispatchEvent(new PopStateEvent('popstate'));
        await wait(() => selected() === 'session-4' && document.querySelector('.terminal-selection-notice'), 'missing URL fallback');
        check(new URL(location.href).searchParams.get('foo') === 'preserved', 'missing URL selection shows notice and normalizes without losing unrelated parameters');
        report.phase = 'complete'; save();
    } catch (error) { report.error = String(error); report.phase = 'failed'; save(); }
}, 0);
'Selection verification started';
