#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { mkdir, rm, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "..");
const stateDir = path.join(repoRoot, ".sonarqube-local");
const statePath = path.join(stateDir, "state.json");
const defaultHostUrl = "http://localhost:9000";
const defaultComposeProject = "sonar-kefapps-zohomail";
const defaultScannerImage = "sonarsource/sonar-scanner-cli:12.1.0.3225_8.0.1";
const tokenName = "terraform-provider-zohomail-local";
const sharedEnvPath =
  process.env.SONAR_SHARED_ENV_PATH ??
  path.resolve(repoRoot, "..", "goose", "keftionnaire", ".git", "ansyo-sonarqube", ".env.local");
const mode = process.argv[2] ?? "bootstrap";

async function main() {
  const sharedState = readSharedState();

  if (mode === "reset") {
    if (!sharedState) {
      run("docker", ["compose", "-p", defaultComposeProject, "down", "-v"], { allowFailure: true });
    }
    await rm(stateDir, { force: true, recursive: true });
    console.log(JSON.stringify({ mode, removed: true, shared: Boolean(sharedState) }, null, 2));
    return;
  }

  if (mode === "status") {
    const state = readState();
    const ps = run("docker", ["compose", "-p", (sharedState?.composeProject ?? defaultComposeProject), "ps", "--format", "json"], { allowFailure: true });
    console.log(JSON.stringify({
      mode,
      composeProject: sharedState?.composeProject ?? defaultComposeProject,
      hostUrl: sharedState?.hostUrl ?? defaultHostUrl,
      running: ps.exitCode === 0,
      shared: Boolean(sharedState),
      state,
    }, null, 2));
    return;
  }

  await mkdir(stateDir, { recursive: true });
  let state;

  if (sharedState && (await sonarIsReady(sharedState.hostUrl))) {
    state = {
      composeProject: sharedState.composeProject,
      hostUrl: sharedState.hostUrl,
      scannerImage: sharedState.scannerImage,
      shared: true,
      token: sharedState.token,
      tokenName: sharedState.tokenName,
      updatedAt: new Date().toISOString(),
    };
  } else {
    run("docker", ["compose", "-p", defaultComposeProject, "up", "-d", "sonarqube_db", "sonarqube"]);
    await waitForSonar(defaultHostUrl);

    const token = await ensureScannerToken(defaultHostUrl);
    state = {
      composeProject: defaultComposeProject,
      hostUrl: defaultHostUrl,
      scannerImage: defaultScannerImage,
      shared: false,
      token,
      tokenName,
      updatedAt: new Date().toISOString(),
    };
  }

  await writeFile(statePath, `${JSON.stringify(state, null, 2)}\n`, "utf8");
  console.log(JSON.stringify(state, null, 2));
}

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: repoRoot,
    encoding: "utf8",
    stdio: options.capture === false ? "inherit" : "pipe",
  });

  if (!options.allowFailure && result.status !== 0) {
    throw new Error(result.stderr || result.stdout || `${command} ${args.join(" ")} failed`);
  }

  return {
    exitCode: result.status ?? 1,
    stderr: result.stderr ?? "",
    stdout: result.stdout ?? "",
  };
}

async function waitForSonar(hostUrl) {
  const deadline = Date.now() + 5 * 60 * 1000;

  while (Date.now() < deadline) {
    if (await sonarIsReady(hostUrl)) {
      return;
    }

    await new Promise((resolve) => setTimeout(resolve, 3000));
  }

  throw new Error("Timed out while waiting for local SonarQube to become ready");
}

async function sonarIsReady(hostUrl) {
  try {
    const response = await fetch(`${hostUrl}/api/system/status`);
    if (!response.ok) {
      return false;
    }

    const payload = await response.json();
    return payload.status === "UP";
  } catch {
    return false;
  }
}

async function ensureScannerToken(hostUrl) {
  const existing = readState();
  if (existing?.token) {
    return existing.token;
  }

  const revokeBody = new URLSearchParams({ name: tokenName });
  await fetch(`${hostUrl}/api/user_tokens/revoke`, {
    body: revokeBody,
    headers: {
      Authorization: `Basic ${Buffer.from("admin:admin").toString("base64")}`,
      "Content-Type": "application/x-www-form-urlencoded",
    },
    method: "POST",
  });

  const generateBody = new URLSearchParams({ name: tokenName, type: "GLOBAL_ANALYSIS_TOKEN" });
  const response = await fetch(`${hostUrl}/api/user_tokens/generate`, {
    body: generateBody,
    headers: {
      Authorization: `Basic ${Buffer.from("admin:admin").toString("base64")}`,
      "Content-Type": "application/x-www-form-urlencoded",
    },
    method: "POST",
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(`Unable to generate SonarQube token: ${message}`);
  }

  const payload = await response.json();
  return payload.token;
}

function readState() {
  if (!existsSync(statePath)) {
    return null;
  }

  return JSON.parse(readFileSync(statePath, "utf8"));
}

function readSharedState() {
  if (!existsSync(sharedEnvPath)) {
    return null;
  }

  const env = parseEnvFile(sharedEnvPath);
  if (!env.SONAR_HOST_URL || !env.SONAR_TOKEN) {
    return null;
  }

  return {
    composeProject: env.SONAR_COMPOSE_PROJECT_NAME || "sonar-kefapps",
    hostUrl: env.SONAR_HOST_URL,
    scannerImage: env.SONAR_SCANNER_IMAGE || defaultScannerImage,
    token: env.SONAR_TOKEN,
    tokenName: env.SONAR_TOKEN_NAME || tokenName,
  };
}

function parseEnvFile(filePath) {
  const env = {};

  for (const line of readFileSync(filePath, "utf8").split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }

    const separator = trimmed.indexOf("=");
    if (separator === -1) {
      continue;
    }

    const key = trimmed.slice(0, separator).trim();
    let value = trimmed.slice(separator + 1).trim();

    if (
      (value.startsWith("\"") && value.endsWith("\"")) ||
      (value.startsWith("'") && value.endsWith("'"))
    ) {
      value = value.slice(1, -1);
    }

    env[key] = value;
  }

  return env;
}

main().catch((error) => {
  console.error(error.message);
  process.exitCode = 1;
});
