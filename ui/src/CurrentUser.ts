import axios, {AxiosError, AxiosResponse} from 'axios';
import * as config from './config';
import {detect} from 'detect-browser';
import {SnackReporter} from './snack/SnackManager';
import {observable, makeObservable, action} from 'mobx';
import {IClient, IUser} from './types';

const tokenKey = 'gotify-login-key';

export class CurrentUser {
    @observable tokenCache: string | null = null;
    @observable reconnectTimeoutId: number | null = null;
    @observable reconnectTime = 7500;
    @observable loggedIn = false;
    @observable refreshKey = 0;
    @observable authKey = 0;
    @observable authenticating = false;
    @observable user: IUser = {name: 'unknown', admin: false, id: -1};
    @observable connectionErrorMessage: string | null = null;

    public constructor(private readonly snack: SnackReporter) {
        this.authenticating = false;
        makeObservable(this);
    }

    @action
    public token = (): string => {
        if (this.tokenCache !== null) {
            return this.tokenCache;
        }

        const localStorageToken = window.localStorage.getItem(tokenKey);
        if (localStorageToken) {
            this.tokenCache = localStorageToken;
            return localStorageToken;
        }

        return '';
    };

    @action
    private readonly setToken = (token: string) => {
        this.tokenCache = token;
        window.localStorage.setItem(tokenKey, token);
    };

    public register = async (name: string, pass: string): Promise<boolean> =>
        axios
            .create()
            .post(config.get('url') + 'user', {name, pass})
            .then(async () => {
                this.snack('User Created. Logging in...');
                await this.login(name, pass);
                return true;
            })
            .catch((error: AxiosError<{error?: string; errorDescription?: string}>) => {
                if (!error || !error.response) {
                    this.snack('No network connection or server unavailable.');
                    return false;
                }
                const {data} = error.response;

                this.snack(
                    `Register failed: ${data?.error ?? 'unknown'}: ${data?.errorDescription ?? ''}`
                );
                return false;
            });

    @action
    public login = async (username: string, password: string) => {
        this.authenticating = true;
        this.connectionErrorMessage = null;
        console.log('[Login] Starting login process for user:', username);

        const browser = detect();
        const name = (browser && browser.name + ' ' + browser.version) || 'unknown browser';
        console.log('[Login] Browser:', name);
        console.log('[Login] API URL:', config.get('url') + 'client');

        return axios
            .create()
            .request({
                url: config.get('url') + 'client',
                method: 'POST',
                data: {name},
                headers: {Authorization: 'Basic ' + btoa(username + ':' + password)},
            })
            .then((resp: AxiosResponse<IClient>) => {
                console.log('[Login] Client creation successful, token received');
                this.snack(`A client named '${name}' was created for your session.`);
                this.setToken(resp.data.token);
                return this.tryAuthenticate();
            })
            .catch((error: AxiosError) => {
                console.error('[Login] Login failed:', error.message);
                console.error('[Login] Response status:', error.response?.status);
                console.error('[Login] Response data:', error.response?.data);

                this.authenticating = false;
                this.loggedIn = false;
                this.tokenCache = null;
                window.localStorage.removeItem(tokenKey);

                if (error.response) {
                    const status = error.response.status;
                    if (status === 401) {
                        this.snack('Invalid username or password');
                    } else if (status === 403) {
                        this.snack('Access forbidden');
                    } else if (status >= 500) {
                        this.snack('Server error: ' + (error.response.statusText || 'Unknown error'));
                    } else {
                        this.snack('Login failed: ' + (error.response.data as any)?.error || error.message);
                    }
                } else if (!error.request) {
                    this.snack('Network error: Request configuration error');
                } else {
                    this.snack('Network error: Server is not reachable. Please check your connection.');
                }

                this.refreshKey++;
                return Promise.reject(error);
            });
    };

    @action
    public tryAuthenticate = async (): Promise<AxiosResponse<IUser>> => {
        this.token();
        if (this.tokenCache === null || this.token() === '') {
            console.log('[Auth] No token available, skipping authentication');
            this.authenticating = false;
            this.loggedIn = false;
            return Promise.reject(new Error('No token'));
        }

        console.log('[Auth] Attempting to authenticate with token:', this.token().substring(0, 10) + '...');

        return axios
            .create()
            .get(config.get('url') + 'current/user', {headers: {'X-Gotify-Key': this.token()}})
            .then((passThrough) => {
                console.log('[Auth] Authentication successful for user:', passThrough.data.name);
                this.user = passThrough.data;
                this.loggedIn = true;
                this.authenticating = false;
                this.connectionErrorMessage = null;
                this.reconnectTime = 7500;
                return passThrough;
            })
            .catch((error: AxiosError) => {
                console.error('[Auth] Authentication failed:', error.message);
                console.error('[Auth] Response status:', error.response?.status);

                this.authenticating = false;

                if (!error || !error.response) {
                    this.connectionError('No network connection or server unavailable.');
                    return Promise.reject(error);
                }

                const status = error.response.status;

                if (status >= 500) {
                    this.connectionError(
                        `${error.response.statusText} (code: ${status}). Server may be temporarily unavailable.`
                    );
                    return Promise.reject(error);
                }

                this.connectionErrorMessage = null;

                if (status >= 400 && status < 500) {
                    this.loggedIn = false;
                    this.tokenCache = null;
                    window.localStorage.removeItem(tokenKey);
                    this.refreshKey++;
                    console.log('[Auth] Invalid credentials, cleared token');
                }
                return Promise.reject(error);
            });
    };

    @action
    public logout = async () => {
        this.loggedIn = false;
        this.refreshKey++;
        await axios
            .get(config.get('url') + 'client')
            .then((resp: AxiosResponse<IClient[]>) => {
                resp.data
                    .filter((client) => client.token === this.tokenCache)
                    .forEach((client) => axios.delete(config.get('url') + 'client/' + client.id));
            })
            .catch(() => Promise.resolve());
        window.localStorage.removeItem(tokenKey);
        this.tokenCache = null;
    };

    public changePassword = (pass: string) => {
        axios
            .post(config.get('url') + 'current/user/password', {pass})
            .then(() => this.snack('Password changed'));
    };

    public tryReconnect = (quiet = false) => {
        this.tryAuthenticate().catch(() => {
            if (!quiet) {
                this.snack('Reconnect failed');
            }
            action(() => {
                this.loggedIn = false;
                this.refreshKey++;
            })();
        });
    };

    @action
    private readonly connectionError = (message: string) => {
        this.connectionErrorMessage = message;
        if (this.reconnectTimeoutId !== null) {
            window.clearTimeout(this.reconnectTimeoutId);
        }
        this.reconnectTimeoutId = window.setTimeout(
            () => this.tryReconnect(true),
            this.reconnectTime
        );
        this.reconnectTime = Math.min(this.reconnectTime * 2, 120000);
    };
}
