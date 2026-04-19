// Debug script to investigate why both worktrees show as selected

console.log("Starting worktree selection debug...");

// Wait for page to load
await new Promise(r => setTimeout(r, 3000));

// Navigate to opencode project
await navigate("/project/opencode", {waitUntil: "networkidle0"});
await new Promise(r => setTimeout(r, 3000));

console.log("Current URL:", page.url());

// Check if Worktrees section is visible
const html = await page.content();
const hasWorktrees = html.includes("Worktrees");
console.log("Has Worktrees section:", hasWorktrees);

if (hasWorktrees) {
    // Extract worktree info from the page
    const worktreeInfo = await page.evaluate(() => {
        const worktreeDivs = document.querySelectorAll('div[style*="border-radius: 8px"]');
        const info = [];
        worktreeDivs.forEach(div => {
            const pathEl = div.querySelector('span[style*="word-break: break-all"]');
            const branchEl = div.querySelector('div[style*="font-size: 12px"]');
            const mainBadge = div.querySelector('span:contains("main")');
            const style = div.getAttribute('style') || '';
            
            // Check if it has green background (selected)
            const isSelected = style.includes('rgba(34, 197, 94') || style.includes('border: 1px solid rgba(34, 197, 94');
            
            info.push({
                path: pathEl ? pathEl.textContent : null,
                branch: branchEl ? branchEl.textContent : null,
                isMain: !!mainBadge,
                isSelected: isSelected,
                style: style.substring(0, 200),
            });
        });
        return info;
    });
    
    console.log("\nWorktree info from page:");
    worktreeInfo.forEach((wt, i) => {
        console.log(`\nWorktree ${i}:`);
        console.log(`  Path: ${wt.path}`);
        console.log(`  Branch: ${wt.branch}`);
        console.log(`  isMain: ${wt.isMain}`);
        console.log(`  isSelected: ${wt.isSelected}`);
    });
}

// Fetch project config and worktrees from API
const projectData = await page.evaluate(async () => {
    try {
        // Get projects
        const projectsRes = await fetch('/api/projects');
        const projects = await projectsRes.json();
        const project = projects.find(p => p.name === 'opencode');
        
        if (!project) return { error: 'Project not found' };
        
        // Get worktrees
        const worktreesRes = await fetch(`/api/review/worktrees?dir=${encodeURIComponent(project.dir)}`);
        const worktrees = await worktreesRes.json();
        
        return {
            projectDir: project.dir,
            worktreesConfig: project.worktrees,
            worktreesFromAPI: worktrees,
        };
    } catch (e) {
        return { error: e.message };
    }
});

console.log("\nProject data from API:");
console.log(JSON.stringify(projectData, null, 2));

// Analyze the logic
console.log("\n\n=== ANALYSIS ===");
console.log("Current URL:", page.url());

// Parse worktree ID from URL
const url = page.url();
const match = url.match(/\/project\/opencode(?:~(\\d+))?/);
const urlWorktreeId = match && match[1] ? parseInt(match[1], 10) : 0;
console.log("URL Worktree ID:", urlWorktreeId);

if (projectData.worktreesFromAPI && projectData.worktreesFromAPI.worktrees) {
    console.log("\nWorktrees from API:");
    projectData.worktreesFromAPI.worktrees.forEach((wt, i) => {
        console.log(`\nWorktree ${i}:`);
        console.log(`  Path: ${wt.path}`);
        console.log(`  isMain: ${wt.isMain}`);
        
        // Calculate ID using same logic as getWorktreeId
        let worktreeId;
        if (wt.isMain) {
            worktreeId = 0;
        } else {
            // Look up in config
            let found = false;
            if (projectData.worktreesConfig) {
                for (const [id, config] of Object.entries(projectData.worktreesConfig)) {
                    if (config.path === wt.path) {
                        worktreeId = parseInt(id, 10);
                        found = true;
                        break;
                    }
                }
            }
            if (!found) {
                worktreeId = 0; // BUG: Falls back to 0!
            }
        }
        
        const isSelected = worktreeId === urlWorktreeId;
        console.log(`  Calculated ID: ${worktreeId}`);
        console.log(`  isSelected: ${isSelected}`);
        
        if (!wt.isMain && !projectData.worktreesConfig) {
            console.log(`  *** BUG: Non-main worktree has no config entry, falls back to ID 0`);
        }
    });
}

console.log("\n\n=== CONCLUSION ===");
console.log("The bug is that getWorktreeId() returns 0 for worktrees without config entries.");
console.log("Since the main worktree also has ID 0, BOTH appear selected (green).");
console.log("Fix: Auto-assign IDs to worktrees without config entries.");

console.log("\nDebug complete!");
