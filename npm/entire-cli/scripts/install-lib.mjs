export const DEFAULT_GITHUB_REPO = "entireio/cli";

function stripLeadingV(version) {
  if (!version) {
    return "";
  }

  return version.startsWith("v") ? version.slice(1) : version;
}

export function normalizeVersion(version) {
  const normalized = stripLeadingV(String(version ?? "").trim());
  if (!normalized || normalized.startsWith("0.0.0")) {
    return null;
  }

  return normalized;
}

export function resolvePlatform(platform = process.platform, arch = process.arch) {
  const resolvedOs =
    platform === "darwin" ? "darwin" : platform === "linux" ? "linux" : null;
  if (!resolvedOs) {
    throw new Error(`unsupported operating system: ${platform}`);
  }

  let resolvedArch;
  switch (arch) {
    case "x64":
    case "amd64":
      resolvedArch = "amd64";
      break;
    case "arm64":
    case "aarch64":
      resolvedArch = "arm64";
      break;
    default:
      throw new Error(`unsupported architecture: ${arch}`);
  }

  return { os: resolvedOs, arch: resolvedArch };
}

export function buildArchiveName(os, arch) {
  return `entire_${os}_${arch}.tar.gz`;
}

export function buildReleaseUrls(repo, version, os, arch) {
  const tag = `v${stripLeadingV(version)}`;
  const archiveName = buildArchiveName(os, arch);
  const base = `https://github.com/${repo}/releases/download/${tag}`;

  return {
    tag,
    archiveName,
    archiveUrl: `${base}/${archiveName}`,
    checksumsUrl: `${base}/checksums.txt`,
  };
}

export function extractChecksum(checksumsText, fileName) {
  const lines = checksumsText.split(/\r?\n/);
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }

    const parts = trimmed.split(/\s+/);
    if (parts.length < 2) {
      continue;
    }

    const checksum = parts[0];
    const candidate = parts.slice(1).join(" ").replace(/^\*/, "");
    if (candidate === fileName) {
      return checksum;
    }
  }

  throw new Error(`checksum not found for ${fileName}`);
}
