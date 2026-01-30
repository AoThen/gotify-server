import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Delete from '@mui/icons-material/Delete';
import Add from '@mui/icons-material/Add';
import Block from '@mui/icons-material/Block';
import Warning from '@mui/icons-material/Warning';
import React from 'react';
import DefaultPage from '../common/DefaultPage';
import Button from '@mui/material/Button';
import TextField from '@mui/material/TextField';
import Dialog from '@mui/material/Dialog';
import DialogTitle from '@mui/material/DialogTitle';
import DialogContent from '@mui/material/DialogContent';
import DialogActions from '@mui/material/DialogActions';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import {observer} from 'mobx-react-lite';
import {useStores} from '../stores';
import {BlockedIPInfo} from './BlacklistStore';
import ConfirmDialog from '../common/ConfirmDialog';

const BlacklistRow: React.FC<{
    info: BlockedIPInfo;
    onUnblock: () => void;
}> = React.memo(({info, onUnblock}) => {
    const isExpired = info.expired;
    const now = new Date();
    const expiresAt = new Date(info.expiresAt);

    return (
        <TableRow sx={{
            opacity: isExpired ? 0.6 : 1,
            backgroundColor: isExpired ? 'action.disabledBackground' : 'inherit'
        }}>
            <TableCell>
                <Box display="flex" alignItems="center" gap={1}>
                    <Block color={isExpired ? "disabled" : "error"} fontSize="small" />
                    <Typography fontFamily="monospace">{info.ip}</Typography>
                </Box>
            </TableCell>
            <TableCell>{info.blockedAt}</TableCell>
            <TableCell>
                {isExpired ? (
                    <Chip label="Expired" color="default" size="small" />
                ) : (
                    <Box>
                        <Chip label="Active" color="error" size="small" />
                        <Typography variant="caption" display="block" sx={{mt: 0.5, color: 'text.secondary'}}>
                            {`${Math.ceil((expiresAt.getTime() - now.getTime()) / 60000)} min remaining`}
                        </Typography>
                    </Box>
                )}
            </TableCell>
            <TableCell>{info.reason}</TableCell>
            <TableCell align="right">
                <IconButton
                    onClick={onUnblock}
                    className="unblock"
                    size="large"
                    disabled={isExpired}
                    title={isExpired ? "Already expired" : "Unblock IP"}
                >
                    <Delete color={isExpired ? "disabled" : "error"} />
                </IconButton>
            </TableCell>
        </TableRow>
    );
});

const WhitelistRow: React.FC<{
    entry: string;
    onRemove: () => void;
}> = React.memo(({entry, onRemove}) => (
    <TableRow>
        <TableCell>
            <Box display="flex" alignItems="center" gap={1}>
                <Warning color="warning" fontSize="small" />
                <Typography fontFamily="monospace">{entry}</Typography>
            </Box>
        </TableCell>
        <TableCell align="right">
            <IconButton onClick={onRemove} className="remove-whitelist" size="large">
                <Delete />
            </IconButton>
        </TableCell>
    </TableRow>
));

const Blacklist = observer(() => {
    const {blacklistStore} = useStores();
    const [unblockIP, setUnblockIP] = React.useState<string>();
    const [showAddWhitelist, setShowAddWhitelist] = React.useState(false);
    const [whitelistEntry, setWhitelistEntry] = React.useState('');
    const [showClearConfirm, setShowClearConfirm] = React.useState(false);

    const activeBlockedCount = blacklistStore.blacklist.blockedIPs?.filter((ip: BlockedIPInfo) => !ip.expired).length ?? 0;
    const expiredBlockedCount = blacklistStore.blacklist.blockedIPs?.filter((ip: BlockedIPInfo) => ip.expired).length ?? 0;

    React.useEffect(() => {
        blacklistStore.refreshBlacklist();
        blacklistStore.refreshWhitelist();
    }, []); // Empty dependency array - only run on mount

    return (
        <DefaultPage
            title="IP Blacklist"
            rightControl={
                <Box display="flex" gap={1}>
                    <Button
                        id="clear-blacklist"
                        variant="outlined"
                        color="warning"
                        onClick={() => setShowClearConfirm(true)}>
                        Clear All
                    </Button>
                    <Button
                        id="add-whitelist"
                        variant="outlined"
                        color="success"
                        startIcon={<Add />}
                        onClick={() => setShowAddWhitelist(true)}>
                        Add to Whitelist
                    </Button>
                </Box>
            }>
            <Grid size={{xs: 12}}>
                <Paper elevation={6} style={{overflowX: 'auto', marginBottom: 16}}>
                    <Box p={2} bgcolor="error.light" borderRadius={1} mb={2}>
                        <Typography variant="subtitle1" fontWeight="bold">
                            Blocked IPs ({activeBlockedCount} active, {expiredBlockedCount} expired)
                        </Typography>
                        <Typography variant="caption">
                            IPs are blocked after 5 failed login attempts within 5 minutes and will be unblocked after 1 hour. Expired entries are kept for reference.
                        </Typography>
                    </Box>
                    <Table id="blacklist-table">
                        <TableHead>
                            <TableRow style={{textAlign: 'center'}}>
                                <TableCell>IP Address</TableCell>
                                <TableCell>Blocked At</TableCell>
                                <TableCell>Status</TableCell>
                                <TableCell>Reason</TableCell>
                                <TableCell />
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {blacklistStore.blacklist.blockedIPs?.length === 0 ? (
                                <TableRow>
                                    <TableCell colSpan={5} align="center">
                                        <Typography color="textSecondary" py={4}>
                                            No blocked IPs
                                        </Typography>
                                    </TableCell>
                                </TableRow>
                            ) : (
                                blacklistStore.blacklist.blockedIPs?.map((info: BlockedIPInfo) => (
                                    <BlacklistRow
                                        key={info.ip}
                                        info={info}
                                        onUnblock={() => setUnblockIP(info.ip)}
                                    />
                                ))
                            )}
                        </TableBody>
                    </Table>
                </Paper>

                <Paper elevation={6} style={{overflowX: 'auto'}}>
                    <Box p={2} bgcolor="warning.light" borderRadius={1} mb={2}>
                        <Typography variant="subtitle1" fontWeight="bold">
                            Whitelist ({blacklistStore.whitelist.count ?? 0})
                        </Typography>
                        <Typography variant="caption">
                            Whitelisted IPs will never be blocked.
                        </Typography>
                    </Box>
                    <Table id="whitelist-table">
                        <TableHead>
                            <TableRow style={{textAlign: 'center'}}>
                                <TableCell>IP / CIDR</TableCell>
                                <TableCell />
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {blacklistStore.whitelist.entries?.length === 0 ? (
                                <TableRow>
                                    <TableCell colSpan={2} align="center">
                                        <Typography color="textSecondary" py={4}>
                                            No whitelist entries
                                        </Typography>
                                    </TableCell>
                                </TableRow>
                            ) : (
                                blacklistStore.whitelist.entries?.map((entry: string) => (
                                    <WhitelistRow
                                        key={entry}
                                        entry={entry}
                                        onRemove={() => blacklistStore.removeFromWhitelist(entry)}
                                    />
                                ))
                            )}
                        </TableBody>
                    </Table>
                </Paper>
            </Grid>

            <Dialog open={!!unblockIP} onClose={() => setUnblockIP(undefined)}>
                <DialogTitle>Confirm Unblock</DialogTitle>
                <DialogContent>
                    <Typography>
                        Are you sure you want to unblock IP {unblockIP}?
                    </Typography>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setUnblockIP(undefined)}>Cancel</Button>
                    <Button
                        onClick={() => {
                            if (unblockIP) {
                                blacklistStore.unblockIP(unblockIP);
                                setUnblockIP(undefined);
                            }
                        }}
                        color="primary">
                        Unblock
                    </Button>
                </DialogActions>
            </Dialog>

            <Dialog open={showAddWhitelist} onClose={() => setShowAddWhitelist(false)}>
                <DialogTitle>Add to Whitelist</DialogTitle>
                <DialogContent>
                    <TextField
                        autoFocus
                        margin="dense"
                        label="IP Address or CIDR (e.g., 192.168.1.0/24)"
                        fullWidth
                        value={whitelistEntry}
                        onChange={(e) => setWhitelistEntry(e.target.value)}
                        placeholder="e.g., 192.168.1.100 or 10.0.0.0/8"
                    />
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setShowAddWhitelist(false)}>Cancel</Button>
                    <Button
                        onClick={() => {
                            if (whitelistEntry) {
                                blacklistStore.addToWhitelist(whitelistEntry);
                                setWhitelistEntry('');
                                setShowAddWhitelist(false);
                            }
                        }}
                        color="primary">
                        Add
                    </Button>
                </DialogActions>
            </Dialog>

            <ConfirmDialog
                open={showClearConfirm}
                title="Clear Blacklist"
                text="Are you sure you want to unblock all IPs? This action cannot be undone."
                fClose={() => setShowClearConfirm(false)}
                fOnSubmit={() => {
                    blacklistStore.clearBlacklist();
                    setShowClearConfirm(false);
                }}
            />
        </DefaultPage>
    );
});

export default Blacklist;
