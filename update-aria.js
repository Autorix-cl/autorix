const fs = require('fs');

const updateFile = (filepath, searchStr, replaceStr, appliedMarker) => {
    let content = fs.readFileSync(filepath, 'utf8');
    if (!content.includes(searchStr)) {
        if (appliedMarker && content.includes(appliedMarker)) {
            return;
        }
        throw new Error(`Expected text not found in ${filepath}`);
    }
    content = content.replace(searchStr, replaceStr);
    fs.writeFileSync(filepath, content, 'utf8');
};

updateFile(
    'console/src/components/resources/resource-table.tsx',
    '<Button variant="ghost" size="icon-sm" onClick={onRefresh} className="h-8 w-8">',
    '<Button variant="ghost" size="icon-sm" onClick={onRefresh} className="h-8 w-8" aria-label="Refresh table">',
    'aria-label="Refresh table"'
);

updateFile(
    'console/src/components/resources/resource-detail-scaffold.tsx',
    '<Button variant="ghost" size="icon-sm" onClick={onBack} className="h-8 w-8">',
    '<Button variant="ghost" size="icon-sm" onClick={onBack} className="h-8 w-8" aria-label="Go back">',
    'aria-label="Go back"'
);

updateFile(
    'console/src/components/ui/code-block.tsx',
    '<Button\n          variant="ghost"\n          size="icon-sm"\n          onClick={handleCopy}\n          className="h-6 w-6 text-muted-foreground hover:text-foreground"\n          title="Copy code"\n        >',
    '<Button\n          variant="ghost"\n          size="icon-sm"\n          onClick={handleCopy}\n          className="h-6 w-6 text-muted-foreground hover:text-foreground"\n          title="Copy code"\n          aria-label="Copy code"\n        >',
    'aria-label="Copy code"'
);

console.log("Updates completed");
