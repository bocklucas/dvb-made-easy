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
      const updated = [...events];

      // When a new step arrives, mark all previous in_progress events as done
      // so they show green checkmarks in the timeline.
      for (let i = 0; i < updated.length; i++) {
        if (updated[i].status === 'in_progress' && updated[i].step !== event.step) {
          updated[i] = { ...updated[i], status: 'done' };
        }
      }

      // For progress updates on the same step (e.g. downloading at 5%, 10%, etc.),
      // update the existing event in-place instead of appending a duplicate.
      const existingIndex = updated.findIndex(
        (e) => e.step === event.step && e.status === 'in_progress' && event.status === 'in_progress'
          && (e.volume ?? '') === (event.volume ?? '')
      );
      if (existingIndex !== -1) {
        updated[existingIndex] = event;
      } else {
        updated.push(event);
      }

      events = updated;
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
