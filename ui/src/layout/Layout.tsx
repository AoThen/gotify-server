import {
    createTheme,
    ThemeProvider,
    StyledEngineProvider,
    Theme,
    useMediaQuery,
} from '@mui/material';
import {makeStyles} from 'tss-react/mui';
import CssBaseline from '@mui/material/CssBaseline';
import * as React from 'react';
import {HashRouter, Navigate, Route, Routes, useNavigate} from 'react-router-dom';
import Header from './Header';
import Navigation from './Navigation';
import ScrollUpButton from '../common/ScrollUpButton';
import SettingsDialog from '../common/SettingsDialog';
import * as config from '../config';
import Applications from '../application/Applications';
import Clients from '../client/Clients';
import Plugins from '../plugin/Plugins';
import Login from '../user/Login';
import Messages from '../message/Messages';
import Users from '../user/Users';
import Blacklist from '../admin/Blacklist';
import {observer} from 'mobx-react-lite';
import {ConnectionErrorBanner} from '../common/ConnectionErrorBanner';
import {useStores} from '../stores';
import {SnackbarProvider} from 'notistack';
import LoadingSpinner from '../common/LoadingSpinner';
import {isThemeKey, ThemeKey} from './theme';

const useStyles = makeStyles()((theme: Theme) => ({
    content: {
        margin: '0 auto',
        marginTop: 64,
        padding: theme.spacing(3),
        width: '100%',
        [theme.breakpoints.down('sm')]: {
            marginTop: 0,
            padding: theme.spacing(1),
        },
    },
}));

const localStorageThemeKey = 'gotify-theme';

const Layout = observer(() => {
    const {currentUser} = useStores();
    const loggedIn = currentUser.loggedIn;
    const authenticating = currentUser.authenticating;
    const user = currentUser.user;
    const connectionErrorMessage = currentUser.connectionErrorMessage;

    const {classes} = useStyles();
    const [currentTheme, setCurrentTheme] = React.useState<ThemeKey>(() => {
        const stored = window.localStorage.getItem(localStorageThemeKey);
        return isThemeKey(stored) ? stored : 'system';
    });
    const [, setTick] = React.useState(0);
    const prefersDark = useMediaQuery('(prefers-color-scheme: dark)');
    const paletteMode = currentTheme === 'system' ? (prefersDark ? 'dark' : 'light') : currentTheme;
    const theme = React.useMemo(
        () =>
            createTheme({
                palette: {
                    mode: paletteMode,
                },
            }),
        [paletteMode]
    );
    const {version} = config.get('version');
    const [navOpen, setNavOpen] = React.useState(false);
    const [showSettings, setShowSettings] = React.useState(false);

    React.useEffect(() => {
        setTick(t => t + 1);
    }, [loggedIn, authenticating]);

    const toggleTheme = () => {
        const nextMap: Record<ThemeKey, ThemeKey> = {
            dark: 'light',
            light: 'system',
            system: 'dark',
        };
        const next = nextMap[currentTheme];
        setCurrentTheme(next);
        localStorage.setItem(localStorageThemeKey, next);
    };

    const authed = (children: React.ReactNode) => (
        <RequireAuth loggedIn={loggedIn} authenticating={authenticating}>
            {children}
        </RequireAuth>
    );

    return (
        <StyledEngineProvider injectFirst>
            <ThemeProvider theme={theme}>
                <HashRouter>
                    {/* This forces all components to fully rerender including useEffects.
                        The refreshKey is updated when store data was cleaned and pages should refetch their data. */}
                    {!connectionErrorMessage ? null : (
                        <ConnectionErrorBanner
                            height={64}
                            retry={() => currentUser.tryReconnect()}
                            message={connectionErrorMessage}
                        />
                    )}
                    <div style={{display: 'flex', flexDirection: 'column'}}>
                        <CssBaseline />
                        <Header
                            admin={user.admin}
                            name={user.name}
                            style={{top: !connectionErrorMessage ? 0 : 64}}
                            version={version}
                            loggedIn={loggedIn}
                            themeMode={currentTheme}
                            toggleTheme={toggleTheme}
                            showSettings={() => setShowSettings(true)}
                            logout={currentUser.logout}
                            setNavOpen={setNavOpen}
                        />
                        <div style={{display: 'flex'}}>
                            <Navigation
                                loggedIn={loggedIn}
                                navOpen={navOpen}
                                setNavOpen={setNavOpen}
                            />
                            <main className={classes.content}>
                                <Routes>
                                    <Route
                                        path="/login"
                                        element={
                                            <LoginPage
                                                loggedIn={loggedIn}
                                                authenticating={authenticating}
                                            />
                                        }
                                    />
                                    <Route path="/" element={authed(<Messages />)} />
                                    <Route
                                        path="/messages/:id"
                                        element={authed(<Messages />)}
                                    />
                                    <Route
                                        path="/applications"
                                        element={authed(<Applications />)}
                                    />
                                    <Route path="/clients" element={authed(<Clients />)} />
                                    <Route path="/users" element={authed(<Users />)} />
                                    <Route path="/blacklist" element={authed(<Blacklist />)} />
                                    <Route path="/plugins" element={authed(<Plugins />)} />
                                    <Route
                                        path="/plugins/:id"
                                        element={authed(
                                            <Lazy
                                                component={() =>
                                                    import('../plugin/PluginDetailView')
                                                }
                                            />
                                        )}
                                    />
                                </Routes>
                            </main>
                        </div>
                        {showSettings && (
                            <SettingsDialog fClose={() => setShowSettings(false)} />
                        )}
                        <ScrollUpButton />
                        <SnackbarProvider />
                    </div>
                </HashRouter>
            </ThemeProvider>
        </StyledEngineProvider>
    );
});

// eslint-disable-next-line
const Lazy = ({component}: {component: () => Promise<{default: React.ComponentType<any>}>}) => {
    const Component = React.lazy(component);

    return (
        <React.Suspense fallback={<LoadingSpinner />}>
            <Component />
        </React.Suspense>
    );
};

const LoginPage: React.FC<{loggedIn: boolean; authenticating: boolean}> = ({
    loggedIn,
    authenticating,
}) => {
    const navigate = useNavigate();

    React.useEffect(() => {
        if (loggedIn && !authenticating) {
            navigate('/applications', {replace: true});
        }
    }, [loggedIn, authenticating, navigate]);

    if (authenticating) {
        return <LoadingSpinner />;
    }

    return <Login />;
};

const RequireAuth: React.FC<
    React.PropsWithChildren<{loggedIn: boolean; authenticating: boolean}>
> = ({children, authenticating, loggedIn}) => {
    if (authenticating) {
        return <LoadingSpinner />;
    }
    if (!loggedIn) {
        return <Navigate replace={true} to="/login" />;
    }
    return <>{children}</>;
};

export default Layout;
