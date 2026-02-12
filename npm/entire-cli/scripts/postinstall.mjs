import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import {
  copyFile,
  mkdtemp,
  readFile,
  rm,
  writeFile,
  chmod,
  mkdir,
  access,
} from "node:fs/promises";
import { constants as fsConstants } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import https from "node:https";
import { fileURLToPath } from "node:url";

import {
  DEFAULT_GITHUB_REPO,
  buildReleaseUrls,
  extractChecksum,
  normalizeVersion,
  resolvePlatform,
} from "./install-lib.mjs";

function log(message) {
  console.log(`[entire-npm] ${message}`);
}

async function pathExists(pathname) {
  try {
    await access(pathname, fsConstants.F_OK);
    return true;
  } catch {
    return false;
  }
}

function buildHeaders() {
  const headers = {
    "User-Agent": "@entireio/cli npm wrapper",
    Accept: "application/vnd.github+json",
  };

  if (process.env.GITHUB_TOKEN) {
    headers.Authorization = `Bearer ${process.env.GITHUB_TOKEN}`;
  }

  return headers;
}

function downloadBuffer(url, headers = {}, redirects = 0) {
  if (redirects > 5) {
    return Promise.reject(new Error(`too many redirects while downloading ${url}`));
  }

  return new Promise((resolve, reject) => {
    const req = https.get(url, { headers }, (res) => {
      const status = res.statusCode ?? 0;

      if ([301, 302, 307, 308].includes(status)) {
        const location = res.headers.location;
        if (!location) {
          reject(new Error(`redirect without location header for ${url}`));
          return;
        }

        res.resume();
        const redirectUrl = new URL(location, url).toString();
        downloadBuffer(redirectUrl, headers, redirects + 1).then(resolve).catch(reject);
        return;
      }

      if (status < 200 || status >= 300) {
        const chunks = [];
        res.on("data", (chunk) => chunks.push(chunk));
        res.on("end", () => {
          const body = Buffer.concat(chunks).toString("utf8").slice(0, 400);
          reject(new Error(`request failed (${status}) for ${url}: ${body}`));
        });
        return;
      }

      const chunks = [];
      res.on("data", (chunk) => chunks.push(chunk));
      res.on("end", () => resolve(Buffer.concat(chunks)));
    });

    req.on("error", (error) => {
      reject(error);
    });
  });
}

async function fetchLatestVersion(repo) {
  const url = `https://api.github.com/repos/${repo}/releases/latest`;
  const body = await downloadBuffer(url, buildHeaders());
  const payload = JSON.parse(body.toString("utf8"));
  const tagName = String(payload.tag_name ?? "").trim();
  const normalized = normalizeVersion(tagName);

  if (!normalized) {
    throw new Error(`invalid latest release tag returned by GitHub: ${tagName}`);
  }

  return normalized;
}

function toAbsoluteHomePath(value, homeDir) {
  if (!value || !homeDir) {
    return value;
  }
  if (value === "$HOME") {
    return homeDir;
  }
  if (value.startsWith("$HOME/")) {
    return path.join(homeDir, value.slice("$HOME/".length));
  }
  if (value.startsWith("~/")) {
    return path.join(homeDir, value.slice(2));
  }
  if (!path.isAbsolute(value)) {
    return path.join(homeDir, value);
  }
  return value;
}

async function goBuild(repoRoot, targetPath) {
  const env = { ...process.env };
  const homeDir = env.HOME || process.env.HOME;
  if (homeDir) {
    env.GOPATH = toAbsoluteHomePath(env.GOPATH || path.join(homeDir, "go"), homeDir);
    env.GOMODCACHE = toAbsoluteHomePath(
      env.GOMODCACHE || path.join(env.GOPATH, "pkg", "mod"),
      homeDir,
    );
    env.GOCACHE = toAbsoluteHomePath(env.GOCACHE || path.join(homeDir, ".cache", "go-build"), homeDir);

    await mkdir(env.GOPATH, { recursive: true });
    await mkdir(env.GOMODCACHE, { recursive: true });
    await mkdir(env.GOCACHE, { recursive: true });
  }

  const build = spawnSync("go", ["build", "-o", targetPath, "./cmd/entire"], {
    cwd: repoRoot,
    encoding: "utf8",
    env,
  });
  if (build.status !== 0) {
    throw new Error(`failed to build from source: ${build.stderr || build.stdout}`);
  }
}

async function installFromSource(packageRoot, repoRoot) {
  const targetPath = path.join(packageRoot, "bin", "entire");
  await mkdir(path.dirname(targetPath), { recursive: true });

  log(`Building Entire from source at ${repoRoot}`);
  await goBuild(repoRoot, targetPath);
  await chmod(targetPath, 0o755);
  log(`Installed binary at ${targetPath}`);
}

async function installFromRelease(packageRoot, repo, version, os, arch) {
  const { tag, archiveName, archiveUrl, checksumsUrl } = buildReleaseUrls(repo, version, os, arch);

  log(`Downloading Entire ${version} for ${os}/${arch} from ${repo} (${tag})`);

  const tmpBase = await mkdtemp(path.join(tmpdir(), "entire-npm-"));
  const archivePath = path.join(tmpBase, archiveName);
  const extractedPath = path.join(tmpBase, "entire");

  try {
    const headers = buildHeaders();
    const archiveBuffer = await downloadBuffer(archiveUrl, headers);
    const checksumsBuffer = await downloadBuffer(checksumsUrl, headers);
    const checksumsText = checksumsBuffer.toString("utf8");

    const expectedChecksum = extractChecksum(checksumsText, archiveName).toLowerCase();
    const actualChecksum = createHash("sha256").update(archiveBuffer).digest("hex").toLowerCase();
    if (actualChecksum !== expectedChecksum) {
      throw new Error(
        `checksum mismatch for ${archiveName}: expected ${expectedChecksum}, got ${actualChecksum}`,
      );
    }

    await writeFile(archivePath, archiveBuffer);

    const extract = spawnSync("tar", ["-xzf", archivePath, "-C", tmpBase, "entire"], {
      encoding: "utf8",
    });
    if (extract.status !== 0) {
      throw new Error(`failed to extract archive: ${extract.stderr || extract.stdout}`);
    }

    const targetPath = path.join(packageRoot, "bin", "entire");
    await mkdir(path.dirname(targetPath), { recursive: true });
    await copyFile(extractedPath, targetPath);
    await chmod(targetPath, 0o755);

    log(`Installed binary at ${targetPath}`);
  } finally {
    await rm(tmpBase, { recursive: true, force: true });
  }
}

async function install() {
  if (process.env.ENTIRE_NPM_SKIP_DOWNLOAD === "1") {
    log("Skipping binary download because ENTIRE_NPM_SKIP_DOWNLOAD=1.");
    return;
  }

  const scriptDir = path.dirname(fileURLToPath(import.meta.url));
  const packageRoot = path.resolve(scriptDir, "..");
  const packageJsonPath = path.join(packageRoot, "package.json");
  const packageJson = JSON.parse(await readFile(packageJsonPath, "utf8"));
  const repoRoot = path.resolve(packageRoot, "..", "..");
  const goModPath = path.join(repoRoot, "go.mod");

  const repo = process.env.ENTIRE_NPM_REPO || DEFAULT_GITHUB_REPO;
  const { os, arch } = resolvePlatform();

  const shouldBuildFromSource =
    process.env.ENTIRE_NPM_BUILD_FROM_SOURCE !== "0" &&
    (await pathExists(goModPath)) &&
    (await pathExists(path.join(repoRoot, "cmd", "entire")));
  if (shouldBuildFromSource) {
    await installFromSource(packageRoot, repoRoot);
    return;
  }

  let version = normalizeVersion(process.env.ENTIRE_NPM_VERSION);
  if (!version) {
    version = normalizeVersion(packageJson.version);
  }
  if (!version) {
    log("Package version is development-only; resolving latest GitHub release tag...");
    version = await fetchLatestVersion(repo);
  }

  await installFromRelease(packageRoot, repo, version, os, arch);
}

install().catch((error) => {
  console.error(`[entire-npm] ${error.message}`);
  process.exit(1);
});
