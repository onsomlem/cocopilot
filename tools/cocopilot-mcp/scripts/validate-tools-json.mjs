import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const repoRoot = path.resolve(__dirname, "..");
const toolsJsonPath = path.join(repoRoot, "tools.json");
const indexPath = path.join(repoRoot, "src", "index.ts");

const toolsJsonRaw = fs.readFileSync(toolsJsonPath, "utf8");
const toolsJson = JSON.parse(toolsJsonRaw);

if (!toolsJson || !Array.isArray(toolsJson.tools)) {
  console.error("tools.json must contain a top-level tools array.");
  process.exit(1);
}

const toolsNames = new Set(
  toolsJson.tools
    .map((tool) => tool?.name)
    .filter((name) => typeof name === "string")
);

if (toolsNames.size === 0) {
  console.error("tools.json tools array is empty or missing names.");
  process.exit(1);
}

const indexSource = fs.readFileSync(indexPath, "utf8");
const handlerRegex = /name\s*===\s*"([^"]+)"/g;
const handlerNames = new Set();

for (const match of indexSource.matchAll(handlerRegex)) {
  const name = match[1];
  if (name.startsWith("coco.")) {
    handlerNames.add(name);
  }
}

const missingHandlers = [...toolsNames].filter((name) => !handlerNames.has(name));
const missingTools = [...handlerNames].filter((name) => !toolsNames.has(name));

if (missingHandlers.length || missingTools.length) {
  if (missingHandlers.length) {
    console.error("tools.json entries missing handlers in src/index.ts:");
    for (const name of missingHandlers) {
      console.error(`- ${name}`);
    }
  }

  if (missingTools.length) {
    console.error("Handlers missing from tools.json:");
    for (const name of missingTools) {
      console.error(`- ${name}`);
    }
  }

  process.exit(1);
}

console.log("tools.json matches MCP handlers in src/index.ts.");
