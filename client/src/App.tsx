import React from 'react';
import { Container, Typography, AppBar, Toolbar, Box } from '@mui/material';
import { styled } from '@mui/material/styles';
import PropertyMap from './components/PropertyMap';
import PropertyStats from './components/PropertyStats';

const StyledContainer = styled(Container)(({ theme }) => ({
    marginTop: theme.spacing(4),
}));

const StyledSection = styled(Box)(({ theme }) => ({
    marginBottom: theme.spacing(4),
}));

function App() {
    return (
        <Box sx={{ flexGrow: 1 }}>
            <AppBar position="static">
                <Toolbar>
                    <Typography variant="h6" sx={{ flexGrow: 1 }}>
                        FundaMental - Amsterdam Property Analysis
                    </Typography>
                </Toolbar>
            </AppBar>

            <StyledContainer>
                <StyledSection>
                    <Typography variant="h4" gutterBottom>
                        Property Statistics
                    </Typography>
                    <PropertyStats />
                </StyledSection>

                <StyledSection>
                    <Typography variant="h4" gutterBottom>
                        Property Map
                    </Typography>
                    <PropertyMap />
                </StyledSection>
            </StyledContainer>
        </Box>
    );
}

export default App; 