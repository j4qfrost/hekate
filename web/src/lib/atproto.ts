// Thin wrapper around @atproto/api for hekate-specific record types. The
// generated types from the lexicons live here; M1 swaps them to real
// codegen output (per ADR 0001).

import type { AtpAgent } from '@atproto/api';

export type AtUri = string;
export type Did = string;
export type Iso8601 = string;

export interface Venue {
  $type: 'app.hekate.venue';
  name: string;
  description?: string;
  geo: { lat: number; lon: number; altitudeMeters?: number };
  address?: {
    text?: string;
    locality?: string;
    region?: string;
    country?: string;
    postalCode?: string;
  };
  capacity?: number;
  amenities?: string[];
  bookingPolicy: 'open' | 'review';
  contact?: { email?: string; url?: string };
  createdAt: Iso8601;
}

export interface Slot {
  $type: 'app.hekate.slot';
  venue: AtUri;
  recurrence?: AtUri;
  start: Iso8601;
  end: Iso8601;
  status: 'open' | 'claimed' | 'cancelled';
  claimedBy?: AtUri;
  notes?: string;
  createdAt: Iso8601;
}

export interface HekateEvent {
  $type: 'app.hekate.event';
  slot: AtUri;
  title: string;
  description?: string;
  tags?: string[];
  visibility?: 'public' | 'unlisted';
  capacityCap?: number;
  externalUrl?: string;
  createdAt: Iso8601;
}

export interface RSVP {
  $type: 'app.hekate.rsvp';
  event: AtUri;
  status: 'going' | 'maybe' | 'declined';
  guestCount?: number;
  note?: string;
  createdAt: Iso8601;
}

// M3 implementation: wraps agent.com.atproto.repo.createRecord and
// agent.com.atproto.repo.deleteRecord with hekate-specific helpers.
export interface HekateAtproto {
  putVenue(agent: AtpAgent, rkey: string, record: Omit<Venue, '$type'>): Promise<AtUri>;
  putSlot(agent: AtpAgent, rkey: string, record: Omit<Slot, '$type'>): Promise<AtUri>;
  putEvent(agent: AtpAgent, rkey: string, record: Omit<HekateEvent, '$type'>): Promise<AtUri>;
  putRSVP(agent: AtpAgent, rkey: string, record: Omit<RSVP, '$type'>): Promise<AtUri>;
}
