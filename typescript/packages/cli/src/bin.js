#!/usr/bin/env node

import { chmodSync, createWriteStream, existsSync, mkdirSync, readFileSync, renameSync, rmSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, join } from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import https from "node:https";

const OWNER = "vectorfy-co";
const REPO = "valbridge";
const ENV_BIN = "VALBRIDGE_CLI_BIN";
const MAX_REDIRECTS = 5;
const REQUEST_TIMEOUT_MS = 30_000;

function packageVersion() {
  const packageJsonPath = join(dirname(fileURLToPath(import.meta.url)), "..", "package.json");
  const pkg = JSON.parse(readFileSync(packageJsonPath, "utf8"));
  return pkg.version;
}

function platformInfo() {
  switch (process.platform) {
    case "darwin":
      if (process.arch === "arm64" || process.arch === "x64") {
        return { platform: "darwin", arch: process.arch, ext: "" };
      }
      break;
    case "linux":
      if (process.arch === "arm64" || process.arch === "x64") {
        return { platform: "linux", arch: process.arch, ext: "" };
      }
      break;
    case "win32":
      if (process.arch === "arm64" || process.arch === "x64") {
        return { platform: "windows", arch: process.arch, ext: ".exe" };
      }
      break;
  }

  throw new Error(`Unsupported platform for valbridge CLI: ${process.platform}/${process.arch}`);
}

function releaseAsset(version) {
  const info = platformInfo();
  const filename = `valbridge-${info.platform}-${info.arch}${info.ext}`;
  const tag = `cli-v${version}`;
  const url = `https://github.com/${OWNER}/${REPO}/releases/download/${tag}/${filename}`;
  return { filename, tag, url };
}

function cachePath(version, filename) {
  return join(homedir(), ".cache", "valbridge", "cli", version, filename);
}

async function download(url, destination, redirects = 0) {
  mkdirSync(dirname(destination), { recursive: true });
  const tempDestination = `${destination}.tmp-${process.pid}-${Date.now()}`;

  try {
    await downloadToFile(url, tempDestination, redirects);
    renameSync(tempDestination, destination);
  } catch (error) {
    rmSync(tempDestination, { force: true });
    throw error;
  }

  if (process.platform !== "win32") {
    chmodSync(destination, 0o755);
  }
}

async function downloadToFile(url, destination, redirects = 0) {
  await new Promise((resolve, reject) => {
    const request = https.get(url, (response) => {
      if (response.statusCode && response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        if (redirects >= MAX_REDIRECTS) {
          response.resume();
          reject(new Error(`Failed to download ${url}: too many redirects`));
          return;
        }
        response.resume();
        downloadToFile(response.headers.location, destination, redirects + 1).then(resolve).catch(reject);
        return;
      }

      if (response.statusCode !== 200) {
        reject(new Error(`Failed to download ${url}: HTTP ${response.statusCode}`));
        response.resume();
        return;
      }

      const file = createWriteStream(destination, { mode: 0o755 });
      response.pipe(file);
      file.on("finish", () => file.close(resolve));
      file.on("error", reject);
    });
    request.setTimeout(REQUEST_TIMEOUT_MS, () => {
      request.destroy(new Error(`Failed to download ${url}: request timed out after ${REQUEST_TIMEOUT_MS}ms`));
    });
    request.on("error", reject);
  });
}

async function resolveBinary() {
  const override = process.env[ENV_BIN];
  if (override) {
    return override;
  }

  const version = packageVersion();
  const { filename, url } = releaseAsset(version);
  const destination = cachePath(version, filename);
  if (!existsSync(destination)) {
    await download(url, destination);
  }
  return destination;
}

const binary = await resolveBinary();
const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}

process.exit(result.status ?? 1);
