import {SnackReporter} from '../snack/SnackManager';
import {CurrentUser} from '../CurrentUser';
import * as config from '../config';
import {AxiosError} from 'axios';
import {IMessage} from '../types';
import {makeObservable, action} from 'mobx';

export class WebSocketStore {
    private wsActive = false;
    private ws: WebSocket | null = null;
    private currentCallback: ((msg: IMessage) => void) | null = null;
    private reconnectTimeout: number | null = null;
    private listenRetryCount = 0;
    private readonly maxListenRetries = 50;

    public constructor(
        private readonly snack: SnackReporter,
        private readonly currentUser: CurrentUser
    ) {
        makeObservable(this, {});
    }

    @action
    public listen = (callback: (msg: IMessage) => void) => {
        if (this.wsActive) {
            return;
        }

        const token = this.currentUser.token();
        if (!token) {
            if (this.listenRetryCount < this.maxListenRetries) {
                this.listenRetryCount++;
                setTimeout(() => this.listen(callback), 100);
            } else {
                console.warn('[WebSocket] Max listen retries reached, stopping retry attempts');
            }
            return;
        }

        this.listenRetryCount = 0;
        this.wsActive = true;
        this.currentCallback = callback;

        const wsUrl = config.get('url').replace('http', 'ws').replace('https', 'wss');
        const ws = new WebSocket(wsUrl + 'stream?token=' + token);

        ws.onerror = (e) => {
            console.log('WebSocket connection errored', e);
            this.handleDisconnect();
        };

        ws.onmessage = (data) => {
            if (this.currentCallback) {
                this.currentCallback(JSON.parse(data.data));
            }
        };

        ws.onclose = () => {
            this.handleDisconnect();
        };

        this.ws = ws;
    };

    @action
    private handleDisconnect = () => {
        this.wsActive = false;
        this.ws = null;

        if (this.reconnectTimeout) {
            window.clearTimeout(this.reconnectTimeout);
        }

        this.reconnectTimeout = window.setTimeout(() => {
            if (this.currentCallback) {
                this.currentUser
                    .tryAuthenticate()
                    .then(() => {
                        this.snack('WebSocket connection closed, trying again in 30 seconds.');
                        this.listen(this.currentCallback!);
                    })
                    .catch((error: AxiosError) => {
                        if (error?.response?.status === 401) {
                            this.snack('Could not authenticate with client token, logging out.');
                        }
                    });
            }
        }, 30000);
    };

    @action
    public close = () => {
        if (this.reconnectTimeout) {
            window.clearTimeout(this.reconnectTimeout);
            this.reconnectTimeout = null;
        }
        this.currentCallback = null;
        if (this.ws) {
            this.ws.close(1000, 'WebSocketStore#close');
            this.ws = null;
        }
        this.wsActive = false;
    };
}
