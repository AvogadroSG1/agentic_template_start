import { Injectable } from '@angular/core';

import { environment } from '../../environments/environment';

export interface HealthStatus {
  status: string;
}

// Transport lives here; components consume signals fed by this service.
// RxJS stays reserved for genuinely event-streamed state.
@Injectable({ providedIn: 'root' })
export class ApiClient {
  async getJson<T>(path: string, init?: RequestInit): Promise<T> {
    const headers = new Headers(init?.headers);
    if (!headers.has('Accept')) {
      headers.set('Accept', 'application/json');
    }
    const response = await fetch(`${environment.apiBaseUrl}${path}`, { ...init, headers });
    if (!response.ok) {
      throw new Error(`GET ${path} failed: ${response.status}`);
    }
    return (await response.json()) as T;
  }

  health(): Promise<HealthStatus> {
    return this.getJson<HealthStatus>('/health');
  }
}
