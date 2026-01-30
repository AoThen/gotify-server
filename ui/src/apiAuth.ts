import axios, {AxiosError} from 'axios';
import {CurrentUser} from './CurrentUser';
import {SnackReporter} from './snack/SnackManager';

export const initAxios = (currentUser: CurrentUser, snack: SnackReporter) => {
    axios.interceptors.request.use((config) => {
        const token = currentUser.token();
        if (token) {
            config.headers['x-gotify-key'] = token;
            console.log('[Axios] Setting token header for:', config.url);
        } else {
            console.log('[Axios] No token available for:', config.url);
        }
        return config;
    });

    axios.interceptors.response.use(undefined, (error: AxiosError) => {
        console.error('[Axios] Response error:', error.message, 'URL:', error.config?.url);

        if (!error.response) {
            console.error('[Axios] No response - network error');
            snack('Network error: Gotify server is not reachable. Please check your connection.');
            return Promise.reject(error);
        }

        const status = error.response.status;

        if (status === 401) {
            console.log('[Axios] 401 Unauthorized, attempting re-authentication');
            currentUser.tryAuthenticate().then(() => {
                snack('Authentication expired, please log in again.');
            }).catch((err) => {
                console.error('[Axios] Re-authentication failed:', err);
                currentUser.refreshKey++;
            });
        }

        if (status === 400 || status === 403 || status === 500) {
            const errorData = error.response.data as {error?: string; errorDescription?: string};
            snack(errorData.error + ': ' + errorData.errorDescription);
        }

        return Promise.reject(error);
    });
};
