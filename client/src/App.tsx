import React from 'react';
import { Container, Typography, AppBar, Toolbar, makeStyles } from '@material-ui/core';
import PropertyMap from './components/PropertyMap';
import PropertyStats from './components/PropertyStats';

const useStyles = makeStyles((theme) => ({
    root: {
        flexGrow: 1,
    },
    title: {
        flexGrow: 1,
    },
    content: {
        marginTop: theme.spacing(4),
    },
    section: {
        marginBottom: theme.spacing(4),
    },
}));

function App() {
    const classes = useStyles();

    return (
        <div className={classes.root}>
            <AppBar position="static">
                <Toolbar>
                    <Typography variant="h6" className={classes.title}>
                        FundaMental - Amsterdam Property Analysis
                    </Typography>
                </Toolbar>
            </AppBar>

            <Container className={classes.content}>
                <div className={classes.section}>
                    <Typography variant="h4" gutterBottom>
                        Property Statistics
                    </Typography>
                    <PropertyStats />
                </div>

                <div className={classes.section}>
                    <Typography variant="h4" gutterBottom>
                        Property Map
                    </Typography>
                    <PropertyMap />
                </div>
            </Container>
        </div>
    );
}

export default App; 