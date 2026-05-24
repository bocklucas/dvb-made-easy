import type { ProgressEvent } from '$lib/types';
import { subscribeRestore } from '$lib/api';

let token = $state<string | null>(null);
let events = $state<ProgressEvent[]>([]);
let connectionLost = $state(false);
let unsubscribe: (() => void) | null = null;

export function getToken() {
  return token;
}

export function getEvents() {
  return events;
}

export function getConnectionLost() {
  return connectionLost;
}

export function subscribe(restoreToken: string) {
  cleanup();
  token = restoreToken;
  events = [];
  connectionLost = false;

  unsubscribe = subscribeRestore(
    restoreToken,
    (event) => {
      events = [...events, event];
    },
    () => {
      connectionLost = true;
    }
  );
}

export function cleanup() {
  if (unsubscribe) {
    unsubscribe();
    unsubscribe = null;
  }
  token = null;
  events = [];
  connectionLost = false;
}
