import Dexie, { Table } from 'dexie';

// 1. Define Types (Mirroring our Go structs)
export interface Room {
  room_id: string;
  name: string;
  last_read_message_id: number; // Server's view of our read status
  unread_count?: number;        // Calculated locally
}

export interface Message {
  room_id: string;
  message_id: number;
  sender_id: string;
  content: any; // Will be JSON (E2EE payload or plain text)
  created_at: string; // ISO String
}

export interface UserConfig {
  key: string;
  value: any;
}

// 2. Define the Database
class ChatDatabase extends Dexie {
  rooms!: Table<Room>;
  messages!: Table<Message>;
  config!: Table<UserConfig>;

  constructor() {
    super('ChatterboxDB');
    
    // Define indexes (Schema)
    this.version(1).stores({
      rooms: 'room_id', // Primary Key
      messages: '[room_id+message_id], room_id, created_at', // Compound PK & Indexes
      config: 'key' // For storing "last_synced_at"
    });
  }
}

export const db = new ChatDatabase();

