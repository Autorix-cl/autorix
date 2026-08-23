import fs from "fs";
import path from "path";
import { RootKeyRecord } from "../schemas/vulcan";

const DB_DIR = path.join(process.cwd(), ".data");
const DB_FILE = path.join(DB_DIR, "vulcan_keys.json");

// Ensure db directory and file exist
if (!fs.existsSync(DB_DIR)) {
  fs.mkdirSync(DB_DIR, { recursive: true });
}

if (!fs.existsSync(DB_FILE)) {
  fs.writeFileSync(DB_FILE, JSON.stringify([]), "utf-8");
}

function readDB(): RootKeyRecord[] {
  try {
    const data = fs.readFileSync(DB_FILE, "utf-8");
    return JSON.parse(data);
  } catch (error) {
    console.error("Failed to read DB", error);
    return [];
  }
}

function writeDB(data: RootKeyRecord[]) {
  fs.writeFileSync(DB_FILE, JSON.stringify(data, null, 2), "utf-8");
}

export const vulcanDb = {
  getKeys: (): RootKeyRecord[] => {
    return readDB();
  },

  getKeyById: (id: string): RootKeyRecord | undefined => {
    return readDB().find((k) => k.id === id);
  },

  addKey: (key: RootKeyRecord) => {
    const keys = readDB();
    keys.push(key);
    writeDB(keys);
  },

  revokeKey: (id: string) => {
    const keys = readDB();
    const index = keys.findIndex((k) => k.id === id);
    if (index !== -1) {
      keys[index].revoked = true;
      writeDB(keys);
      return true;
    }
    return false;
  },
};
