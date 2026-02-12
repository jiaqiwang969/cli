import test from "node:test";
import assert from "node:assert/strict";

import {
  buildArchiveName,
  buildReleaseUrls,
  extractChecksum,
  normalizeVersion,
  resolvePlatform,
} from "../scripts/install-lib.mjs";

test("normalizeVersion strips leading v", () => {
  assert.equal(normalizeVersion("v1.2.3"), "1.2.3");
});

test("normalizeVersion returns null for development versions", () => {
  assert.equal(normalizeVersion("0.0.0-dev"), null);
  assert.equal(normalizeVersion("0.0.0"), null);
});

test("resolvePlatform maps supported platform and architecture", () => {
  assert.deepEqual(resolvePlatform("darwin", "x64"), { os: "darwin", arch: "amd64" });
  assert.deepEqual(resolvePlatform("linux", "arm64"), { os: "linux", arch: "arm64" });
});

test("resolvePlatform rejects unsupported platforms", () => {
  assert.throws(() => resolvePlatform("win32", "x64"), /unsupported operating system/);
  assert.throws(() => resolvePlatform("linux", "ppc64"), /unsupported architecture/);
});

test("buildArchiveName and buildReleaseUrls produce expected values", () => {
  assert.equal(buildArchiveName("linux", "amd64"), "entire_linux_amd64.tar.gz");

  const urls = buildReleaseUrls("entireio/cli", "1.2.3", "darwin", "arm64");
  assert.equal(urls.tag, "v1.2.3");
  assert.equal(urls.archiveName, "entire_darwin_arm64.tar.gz");
  assert.equal(
    urls.archiveUrl,
    "https://github.com/entireio/cli/releases/download/v1.2.3/entire_darwin_arm64.tar.gz",
  );
  assert.equal(
    urls.checksumsUrl,
    "https://github.com/entireio/cli/releases/download/v1.2.3/checksums.txt",
  );
});

test("extractChecksum finds the correct checksum", () => {
  const checksums = [
    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  entire_linux_amd64.tar.gz",
    "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  entire_darwin_arm64.tar.gz",
  ].join("\n");

  assert.equal(
    extractChecksum(checksums, "entire_darwin_arm64.tar.gz"),
    "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
  );
  assert.throws(() => extractChecksum(checksums, "entire_linux_arm64.tar.gz"), /checksum not found/);
});
