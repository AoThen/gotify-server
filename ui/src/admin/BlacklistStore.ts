import axios from 'axios';
import * as config from '../config';
import {action, makeObservable, observable} from 'mobx';
import {SnackReporter} from '../snack/SnackManager';

export interface BlockedIPInfo {
    ip: string;
    blockedAt: string;
    expiresAt: string;
    reason: string;
    expired: boolean;
}

export interface BlacklistList {
    blockedCount: number;
    blockedIPs: BlockedIPInfo[];
}

export interface WhitelistInfo {
    entries: string[];
    count: number;
}

export class BlacklistStore {
    @observable blacklist: BlacklistList = {blockedCount: 0, blockedIPs: []};
    @observable whitelist: WhitelistInfo = {entries: [], count: 0};
    @observable loading = false;

    public constructor(private readonly snack: SnackReporter) {
        makeObservable(this);
    }

    @action
    public refreshBlacklist = async (): Promise<void> => {
        this.loading = true;
        try {
            const response = await axios.get<BlacklistList>(`${config.get('url')}admin/blacklist`);
            this.blacklist = response.data;
        } catch (_error) {
            this.snack('Failed to load blacklist');
        } finally {
            this.loading = false;
        }
    };

    @action
    public refreshWhitelist = async (): Promise<void> => {
        try {
            const response = await axios.get<WhitelistInfo>(`${config.get('url')}admin/whitelist`);
            this.whitelist = response.data;
        } catch (_error) {
            this.snack('Failed to load whitelist');
        }
    };

    public unblockIP = async (ip: string): Promise<void> => {
        try {
            await axios.delete(`${config.get('url')}admin/blacklist/${ip}`);
            this.snack(`IP ${ip} unblocked successfully`);
            await this.refreshBlacklist();
        } catch (_error) {
            this.snack('Failed to unblock IP');
        }
    };

    public clearBlacklist = async (): Promise<void> => {
        try {
            await axios.post(`${config.get('url')}admin/blacklist/clear-all`);
            this.snack('Blacklist cleared successfully');
            await this.refreshBlacklist();
        } catch (_error) {
            this.snack('Failed to clear blacklist');
        }
    };

    public addToWhitelist = async (entry: string): Promise<void> => {
        try {
            await axios.post(`${config.get('url')}admin/whitelist`, {entry});
            this.snack('Entry added to whitelist');
            await this.refreshWhitelist();
        } catch (_error) {
            this.snack('Failed to add to whitelist');
        }
    };

    public removeFromWhitelist = async (entry: string): Promise<void> => {
        try {
            await axios.delete(`${config.get('url')}admin/whitelist/${entry}`);
            this.snack('Entry removed from whitelist');
            await this.refreshWhitelist();
        } catch (_error) {
            this.snack('Failed to remove from whitelist');
        }
    };
}
