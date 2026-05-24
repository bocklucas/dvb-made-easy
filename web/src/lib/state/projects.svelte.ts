import type { Project } from '$lib/types';
import * as api from '$lib/api';

let projects = $state<Project[]>([]);
let selectedId = $state<string | null>(null);

export function getProjects() {
  return projects;
}

export function getSelectedId() {
  return selectedId;
}

export function setSelectedId(id: string | null) {
  selectedId = id;
}

export async function load() {
  projects = await api.listProjects();
}

export async function remove(id: string) {
  await api.deleteProject(id);
  projects = projects.filter((p) => p.id !== id);
  if (selectedId === id) {
    selectedId = projects.length > 0 ? projects[0].id : null;
  }
}
