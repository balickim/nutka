// Exposes the independent teacher session facade and its role-specific API lifecycle.

import { createAuthRealm } from "./realm";
import { classifyAuthError } from "./realm";

export type Teacher = {
  id: string;
  email: string;
  name?: string;
  display_name?: string;
  timezone?: string;
  [key: string]: unknown;
};
export type MeResult = Awaited<ReturnType<typeof teacherAuth.fetchMe>>;
export type AuthState = ReturnType<typeof teacherAuth.getState>;

export const teacherAuth = createAuthRealm<Teacher>({
  collection: "teachers",
  mePath: "/api/teachers/auth/me",
  logoutPath: "/api/teachers/auth/logout",
  channelName: "nutka-teacher-auth",
  displayName: (record) => record.name || record.display_name || record.email,
});

export { classifyAuthError };
export const subscribe = teacherAuth.subscribe;
export const getAuthState = teacherAuth.getState;
export const getTeacherDisplayName = teacherAuth.getDisplayName;
export const fetchAuthMe = teacherAuth.fetchMe;
export const bootstrapAuth = teacherAuth.bootstrap;
export const login = teacherAuth.login;
export const logout = teacherAuth.logout;
