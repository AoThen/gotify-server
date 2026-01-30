import {reaction, IReactionDisposer} from 'mobx';
import * as Notifications from './snack/browserNotification';
import {StoreMapping} from './stores';

const AUDIO_REPEAT_DELAY = 1000;

export const registerReactions = (stores: StoreMapping) => {
    const executePendingDeletes = stores.messagesStore.executePendingDeletes;
    window.addEventListener('pagehide', executePendingDeletes);
    window.addEventListener('beforeunload', executePendingDeletes);

    let audio: HTMLAudioElement | undefined;
    let lastAudio = 0;
    let wsListenerRegistered = false;

    const clearAll = () => {
        stores.messagesStore.clearAll();
        stores.appStore.clear();
        stores.clientStore.clear();
        stores.userStore.clear();
        try {
            stores.wsStore.close();
        } catch (_e) {
            // ignore close errors
            void _e;
        }
        wsListenerRegistered = false;
    };

    const loadAll = () => {
        if (wsListenerRegistered) {
            return;
        }
        wsListenerRegistered = true;

        stores.wsStore.listen((message) => {
            stores.messagesStore.publishSingleMessage(message);
            Notifications.notifyNewMessage(message);
            if (message.priority >= 4 && Date.now() > lastAudio + AUDIO_REPEAT_DELAY) {
                lastAudio = Date.now();

                audio ??= new Audio('static/notification.ogg');
                audio.currentTime = 0;
                audio.play().catch(() => {
                    // Audio autoplay may be blocked by browser
                });
            }
        });
        stores.appStore.refresh();
    };

    const disposePageHide = () => {
        window.removeEventListener('pagehide', executePendingDeletes);
        window.removeEventListener('beforeunload', executePendingDeletes);
    };

    const reactionDisposers: IReactionDisposer[] = [];

    reactionDisposers.push(
        reaction(
            () => stores.currentUser.loggedIn,
            (loggedIn) => {
                if (loggedIn) {
                    loadAll();
                    stores.currentUser.refreshKey++;
                } else {
                    clearAll();
                }
            }
        )
    );

    reactionDisposers.push(
        reaction(
            () => stores.currentUser.connectionErrorMessage,
            (connectionErrorMessage) => {
                if (!connectionErrorMessage) {
                    clearAll();
                    loadAll();
                    stores.currentUser.refreshKey++;
                }
            }
        )
    );

    return () => {
        reactionDisposers.forEach(disposer => disposer());
        disposePageHide();
    };
};
