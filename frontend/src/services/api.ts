import axios from 'axios';

const API_BASE = 'http://localhost:8081';

export interface RenderOptions {
  engine: string;
  use_cpu: boolean;
  cycles_device: string;
  threads: number;
  format: string;
  scene: string;
  background: boolean;
}

export type Status = 'CREATED' | 'PENDING_METADATA' | 'PROCESSING' | 'COMPLETED' | 'FAILED' | 'STITCHING';

export interface Job {
  id: string;
  name: string;
  s3_path: string;
  frame_start: number;
  frame_end: number;
  worker_count: number;
  status: Status;
  result_path?: string;
  preview_url?: string;
  options: RenderOptions;
  created_at: string;
  updated_at: string;
}

export interface Task {
  id: string;
  job_id: string;
  frame_start: number;
  frame_end: number;
  status: Status;
  worker_id?: string;
  result_path?: string;
  created_at: string;
  updated_at: string;
}

export interface JobResponse {
  job: Job;
  tasks: Task[];
}

export interface CreateJobRequest {
  name: string;
  s3_path: string;
  frame_start: number;
  frame_end: number;
  worker_count: number;
  options: RenderOptions;
  preview_url?: string;
}

const api = axios.create({
  baseURL: API_BASE,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const jobService = {
  listJobs: async (): Promise<Job[]> => {
    const { data } = await api.get<Job[]>('/jobs');
    return data;
  },
  getJob: async (id: string): Promise<JobResponse> => {
    const { data } = await api.get<JobResponse>(`/jobs/${id}`);
    return data;
  },
  createJob: async (req: CreateJobRequest): Promise<Job> => {
    const { data } = await api.post<Job>('/jobs', req);
    return data;
  },
  getUploadUrl: async (filename: string): Promise<string> => {
    // Stubbed as requested, will move to api-gateway
    console.log('Stub: Getting upload URL for', filename);
    return `http://localhost:9000/bucket/${filename}?stub=true`;
  }
};

export default api;
