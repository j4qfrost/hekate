// REST client for hekate-server. Read-only — writes go directly to the
// user's PDS via $lib/atproto.

const DEFAULT_API_URL =
  typeof process !== 'undefined' && process.env?.PUBLIC_HEKATE_API_URL
    ? process.env.PUBLIC_HEKATE_API_URL
    : 'http://localhost:8080';

export interface VenueSummary {
  did: string;
  rkey: string;
  name: string;
  geo: { lat: number; lon: number };
  amenities: string[];
  bookingPolicy: 'open' | 'review';
}

export interface SlotSummary {
  did: string;
  rkey: string;
  venueUri: string;
  start: string;
  end: string;
  status: 'open' | 'claimed' | 'cancelled';
}

export class HekateClient {
  constructor(private readonly baseUrl: string = DEFAULT_API_URL) {}

  async healthz(): Promise<boolean> {
    const res = await fetch(`${this.baseUrl}/healthz`);
    if (!res.ok) return false;
    const body = (await res.json()) as { ok: boolean };
    return body.ok === true;
  }

  // M1: real implementations land when the server's /venues, /slots, /events
  // endpoints stop returning 501.
  async venuesNear(_bbox: [number, number, number, number]): Promise<VenueSummary[]> {
    return [];
  }

  async slotsForVenue(_venueUri: string): Promise<SlotSummary[]> {
    return [];
  }
}

export const hekate = new HekateClient();
