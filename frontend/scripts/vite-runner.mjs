import { webcrypto } from "node:crypto";
import { pathToFileURL } from "node:url";
import { resolve } from "node:path";

if (!globalThis.crypto || typeof globalThis.crypto.getRandomValues !== "function") {
  globalThis.crypto = webcrypto;
}

process.argv[1] = resolve("node_modules/vite/bin/vite.js");

await import(pathToFileURL(process.argv[1]).href);
