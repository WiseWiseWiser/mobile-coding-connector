export const BlockDestructiveCommands = async ({ project, client }) => {
  // Define patterns for destructive commands
  // Using (^|[;|&]|\$\(|\`\s*) before command to match at start or after separators
  const blockedPatterns = [
    // Process termination
    { pattern: /(^|[;|&]|\$\(|\`\s*)kill\s+/, name: "kill command" },
    { pattern: /(^|[;|&]|\$\(|\`\s*)killall\s+/, name: "killall command" },
    { pattern: /(^|[;|&]|\$\(|\`\s*)pkill\s+/, name: "pkill command" },
    
    // Git destructive operations
    { pattern: /(^|[;|&]|\$\(|\`\s*)git\s+reset\s+(--hard|\s)/, name: "git reset" },
    { pattern: /(^|[;|&]|\$\(|\`\s*)git\s+checkout/, name: "git checkout (all forms blocked)" },
    { pattern: /(^|[;|&]|\$\(|\`\s*)git\s+clean\s+(-f|-xfd)/, name: "git clean (destructive)" },
    { pattern: /(^|[;|&]|\$\(|\`\s*)git\s+push\s+--force-with-lease/, name: "git push --force-with-lease" },
    { pattern: /(^|[;|&]|\$\(|\`\s*)git\s+push\s+(-f|--force)/, name: "git push --force" },
    { pattern: /(^|[;|&]|\$\(|\`\s*)git\s+branch\s+-D/, name: "git branch -D (force delete)" },
    { pattern: /(^|[;|&]|\$\(|\`\s*)git\s+rebase\s+--abort/, name: "git rebase --abort" },
    
    // File system destructive operations
    { pattern: /(^|[;|&]|\$\(|\`\s*)rm\s+-rf?\s+/, name: "rm -rf (recursive delete)" },
    { pattern: /(^|[;|&]|\$\(|\`\s*)rm\s+--recursive/, name: "rm --recursive" },
    { pattern: /(^|[;|&]|\$\(|\`\s*)rmdir\s+/, name: "rmdir" },
    { pattern: /(^|[;|&]|\$\(|\`\s*)dd\s+.*of=\//, name: "dd with output to device" },
    { pattern: /(^|[;|&]|\$\(|\`\s*)mkfs\./, name: "mkfs (format filesystem)" },
    { pattern: /(^|[;|&]|\$\(|\`\s*)fdisk\s+/, name: "fdisk" },
    { pattern: /(^|[;|&]|\$\(|\`\s*)mv\s+.*\s+\/dev\/null/, name: "mv to /dev/null" },
    
    // Output redirection to null that could hide errors
    { pattern: /2?>\s*\/dev\/null.*$/, name: "redirecting output to /dev/null" },
    
    // History manipulation
    { pattern: /(^|[;|&]|\$\(|\`\s*)history\s+-c/, name: "history -c (clear history)" },
  ];

  // Commands that are always allowed despite matching patterns
  // These patterns use \b (word boundary) to match complete commands
  const allowedPatterns = [
    // Safe git operations
    /^git\s+status\b/,
    /^git\s+add\s+/,
    /^git\s+commit/,
    /^git\s+log/,
    /^git\s+diff/,
    /^git\s+branch\s*(?!-D\b)/,  // branch commands except -D
    /^git\s+checkout\s+(?!-f\b)/, // checkout without -f
    /^git\s+push\s+(?!-(f|-force)\b)/, // push without --force
    /^git\s+pull/,
    /^git\s+fetch/,
    /^git\s+merge/,
    /^git\s+rebase\s+(?!--abort\b)/,
    /^git\s+stash/,
    /^git\s+tag/,
    /^git\s+show/,
    /^git\s+remote/,
    /^git\s+config/,
    /^git\s+clone/,
    /^git\s+init/,
    
    // Safe read-only commands
    /^ls\b/,
    /^cat\b/,
    /^find\s+/,
    /^grep\s+/,
    /^head\s+/,
    /^tail\s+/,
    /^less\s+/,
    /^more\s+/,
    /^wc\s+/,
    /^sort\s+/,
    /^uniq\s+/,
    /^awk\s+/,
    /^sed\s+/,
    /^cut\s+/,
    /^tr\s+/,
    /^echo\s+/,
    /^printf\s+/,
    /^pwd\b/,
    /^which\s+/,
    /^whoami\b/,
    /^id\b/,
    /^uname\b/,
    /^env\b/,
    /^printenv\b/,
    /^date\b/,
    /^time\b/,
    /^cal\b/,
    
    // Build/test commands
    /^npm\s+/,
    /^yarn\s+/,
    /^pnpm\s+/,
    /^bun\s+/,
    /^node\s+/,
    /^python\s+/,
    /^python3\s+/,
    /^pip\s+/,
    /^pip3\s+/,
    /^go\s+/,
    /^cargo\s+/,
    /^rustc\s+/,
    /^make\s+/,
    /^cmake\s+/,
    /^docker\s+/,
    /^kubectl\s+/,
    /^terraform\s+/,
    /^ansible/,
    /^pytest\s+/,
    /^jest\s+/,
    /^vitest\s+/,
    /^mocha\s+/,
    /^cypress\s+/,
    /^playwright\s+/,
    /^curl\s+/,
    /^wget\s+/,
    
    // Version control (non-git)
    /^hg\s+/,  // Mercurial
    /^svn\s+/, // Subversion
  ];

  return {
    "tool.execute.before": async (input, output) => {
      if (input.tool === "bash") {
        const command = output.args.command.trim();
        
        // First check if command is explicitly allowed
        for (const pattern of allowedPatterns) {
          if (pattern.test(command)) {
            return; // Command is safe, allow it
          }
        }
        
        // Check against blocked patterns
        for (const { pattern, name } of blockedPatterns) {
          if (pattern.test(command)) {
            // Double-check it's not in allowed list (for edge cases)
            for (const allowedPattern of allowedPatterns) {
              if (allowedPattern.test(command)) {
                return; // Actually allowed
              }
            }
            
            throw new Error(
              `🚫 BLOCKED: Destructive command detected: "${command}"\n\n` +
              `Type: ${name}\n\n` +
              `This command has been blocked by the BlockDestructiveCommands plugin for safety reasons.\n\n` +
              `If you genuinely need to run this command:\n` +
              `1. Use your terminal directly outside of opencode\n` +
              `2. Or temporarily disable the plugin by removing it from .opencode/plugins/\n\n` +
              `⚠️  Warning: This command could result in data loss or system instability.`
            );
          }
        }
      }
    },
  };
};
