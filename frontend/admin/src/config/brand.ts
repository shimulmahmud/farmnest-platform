/** Shop name shown in the admin panel. Set VITE_APP_NAME when building to change it. */
export const APP_NAME: string = (import.meta.env.VITE_APP_NAME as string | undefined)?.trim() || 'FarmNest';
