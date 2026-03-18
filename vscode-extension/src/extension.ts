import * as vscode from "vscode";
import * as fs from "fs";
import * as path from "path";

interface LaunchRequest {
  task_id: string;
  task_type: string;
  priority: string;
  status: string;
  worktree_path: string;
  branch: string;
  resume: boolean;
  timestamp: string;
}

const TYPE_ICONS: Record<string, string> = {
  bug: "bug",
  feat: "add",
  spike: "beaker",
  refactor: "wrench",
};

const PRIORITY_COLORS: Record<string, string> = {
  P0: "terminal.ansiRed",
  P1: "terminal.ansiYellow",
  P2: "terminal.ansiCyan",
  P3: "terminal.ansiWhite",
};

export function activate(context: vscode.ExtensionContext): void {
  const homeDir = process.env.HOME || "";
  if (!homeDir) {
    return;
  }

  const launchFile = path.join(homeDir, ".adb_terminal_launch.json");

  // Watch for launch requests
  const watcher = fs.watch(homeDir, (eventType, filename) => {
    if (filename === ".adb_terminal_launch.json") {
      handleLaunchRequest(launchFile);
    }
  });

  context.subscriptions.push({ dispose: () => watcher.close() });

  // Check if there's already a pending request on activation
  if (fs.existsSync(launchFile)) {
    handleLaunchRequest(launchFile);
  }
}

function handleLaunchRequest(launchFile: string): void {
  let data: string;
  try {
    data = fs.readFileSync(launchFile, "utf8");
  } catch {
    return;
  }

  let req: LaunchRequest;
  try {
    req = JSON.parse(data);
  } catch {
    return;
  }

  // Ignore stale requests (> 5 seconds old)
  const age = Date.now() - new Date(req.timestamp).getTime();
  if (age > 5000) {
    return;
  }

  // Delete the file immediately to prevent re-processing
  try {
    fs.unlinkSync(launchFile);
  } catch {
    // ignore
  }

  // Build terminal options
  const iconName = TYPE_ICONS[req.task_type] || "tasklist";
  const colorName = PRIORITY_COLORS[req.priority] || "terminal.ansiCyan";

  const args = ["--dangerously-skip-permissions"];
  if (req.resume) {
    args.push("--continue");
  }

  const terminal = vscode.window.createTerminal({
    name: `${req.task_id} ${req.task_type} ${req.priority}`,
    iconPath: new vscode.ThemeIcon(iconName),
    color: new vscode.ThemeColor(colorName),
    cwd: req.worktree_path,
  });

  terminal.show();
  terminal.sendText(`claude ${args.join(" ")}`);
}

export function deactivate(): void {}
