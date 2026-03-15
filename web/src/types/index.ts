export interface LogItem {
  timestamp: number;
  level: number;
  service: string;
  host: string;
  message: string;
}

export interface Stats {
  ingestion_rate: number;
  disk_usage: number;
  total_logs: number;
  level_dist?: Record<string, number>;
  top_services?: Record<string, number>;
}

export interface Instance {
  instance_id: string;
  service_name: string;
  hostname: string;
  ip: string;
  sdk_version: string;
  language: string;
  registered_at: number;
  last_seen_at: number;
}

export interface User {
  username: string;
  role: string;
  created_at?: string;
}

export interface ApiKey {
  id: string;
  name: string;
  prefix: string;
  type: string;
  created_at: string;
  created_by: string;
}

export interface SystemStatus {
  initialized: boolean;
  node_role: 'all' | 'admin' | 'engine';
  version: string;
}
