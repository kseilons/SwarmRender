import { UserManager, WebStorageStateStore } from "oidc-client-ts";

const authority = import.meta.env.VITE_AUTH_AUTHORITY || "http://localhost:8083/realms/render-farm";
const clientId = import.meta.env.VITE_AUTH_CLIENT_ID || "render-farm-frontend";

export const userManager = new UserManager({
  authority: authority,
  client_id: clientId,
  redirect_uri: window.location.origin,
  userStore: new WebStorageStateStore({ store: window.sessionStorage }),
  onSigninCallback: () => {
    window.history.replaceState({}, document.title, window.location.pathname);
  },
});

export const getAccessToken = async (): Promise<string | null> => {
  try {
    const user = await userManager.getUser();
    return user?.access_token || null;
  } catch (err) {
    console.error("Failed to get user from OIDC", err);
    return null;
  }
};
