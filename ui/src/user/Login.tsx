import Button from '@mui/material/Button';
import Grid from '@mui/material/Grid';
import TextField from '@mui/material/TextField';
import React from 'react';
import Container from '../common/Container';
import DefaultPage from '../common/DefaultPage';
import * as config from '../config';
import RegistrationDialog from './Register';
import {useStores} from '../stores';
import {observer} from 'mobx-react-lite';
import logger from '../utils/logger';

const Login = observer(() => {
    const [username, setUsername] = React.useState('');
    const [password, setPassword] = React.useState('');
    const [registerDialog, setRegisterDialog] = React.useState(false);
    const {currentUser} = useStores();

    const canSubmit = React.useMemo(() => {
        return (
            username.trim().length > 0 &&
            password.length > 0 &&
            !currentUser.authenticating
        );
    }, [username, password, currentUser.authenticating]);

    const registerButton = () => {
        if (config.get('register'))
            return (
                <Button
                    id="register"
                    variant="contained"
                    color="primary"
                    onClick={() => setRegisterDialog(true)}>
                    Register
                </Button>
            );
        else return null;
    };
    const login = (e: React.MouseEvent<HTMLButtonElement>) => {
        e.preventDefault();
        if (canSubmit) {
            logger.log('[Login] Submitting login form');
            currentUser.login(username, password);
        } else {
            logger.log('[Login] Cannot submit - username or password empty');
        }
    };
    return (
        <DefaultPage title="Login" rightControl={registerButton()} maxWidth={250}>
            <Grid size={{xs: 12}} style={{textAlign: 'center'}}>
                <Container>
                    <form onSubmit={(e) => e.preventDefault()} id="login-form">
                        <TextField
                            autoFocus
                            id="username"
                            className="name"
                            label="Username"
                            name="username"
                            margin="dense"
                            autoComplete="username"
                            value={username}
                            onChange={(e) => setUsername(e.target.value)}
                            disabled={currentUser.authenticating}
                        />
                        <TextField
                            id="password"
                            type="password"
                            className="password"
                            label="Password"
                            name="password"
                            margin="normal"
                            autoComplete="current-password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            disabled={currentUser.authenticating}
                        />
                        <Button
                            type="submit"
                            variant="contained"
                            size="large"
                            className="login"
                            color="primary"
                            disabled={!canSubmit}
                            style={{marginTop: 15, marginBottom: 5}}
                            loading={currentUser.authenticating}
                            onClick={login}>
                            Login
                        </Button>
                    </form>
                </Container>
            </Grid>
            {registerDialog && (
                <RegistrationDialog
                    fClose={() => setRegisterDialog(false)}
                    fOnSubmit={currentUser.register}
                />
            )}
        </DefaultPage>
    );
});

export default Login;
