const fs = require('fs');

const updateFile = (filepath, searchStr, replaceStr) => {
    let content = fs.readFileSync(filepath, 'utf8');
    if (!content.includes(searchStr)) {
        throw new Error(`Expected text not found in ${filepath}`);
    }
    content = content.replace(searchStr, replaceStr);
    fs.writeFileSync(filepath, content, 'utf8');
};

updateFile(
    'console/src/components/layout/command-palette.tsx',
    'className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-xs font-medium text-foreground hover:bg-primary/10 hover:text-primary transition-colors text-left"',
    'aria-label={`Navigate to ${item.label}`}\n                      className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-xs font-medium text-foreground hover:bg-primary/10 hover:text-primary transition-colors text-left"'
);

// We need a separate pass for the second one
let content = fs.readFileSync('console/src/components/layout/command-palette.tsx', 'utf8');
const secondSearch = 'onClick={() => handleSelectAction(act.action)}\n                      className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-xs font-medium text-foreground hover:bg-primary/10 hover:text-primary transition-colors text-left"';
if (!content.includes(secondSearch)) {
    throw new Error('Expected action button text not found in console/src/components/layout/command-palette.tsx');
}
content = content.replace(
    secondSearch,
    'onClick={() => handleSelectAction(act.action)}\n                      aria-label={act.label}\n                      className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-xs font-medium text-foreground hover:bg-primary/10 hover:text-primary transition-colors text-left"'
);
fs.writeFileSync('console/src/components/layout/command-palette.tsx', content, 'utf8');

updateFile(
    'console/src/components/layout/command-dialog.tsx',
    'className="flex w-full items-center justify-between rounded-lg p-2.5 text-left text-xs transition-colors hover:bg-accent hover:text-accent-foreground group cursor-pointer"',
    'aria-label={`Navigate to ${item.title}`}\n                  className="flex w-full items-center justify-between rounded-lg p-2.5 text-left text-xs transition-colors hover:bg-accent hover:text-accent-foreground group cursor-pointer"'
);

console.log("Updates completed");
