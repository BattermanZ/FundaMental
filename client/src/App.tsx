import React from 'react';
import { Container, Typography, AppBar, Toolbar, Box, Tabs, Tab } from '@mui/material';
import { styled } from '@mui/material/styles';
import { BrowserRouter as Router, Routes, Route, Link, useLocation, useNavigate } from 'react-router-dom';
import PropertyMap from './components/PropertyMap';
import PropertyStats from './components/PropertyStats';
import PropertyCharts from './components/PropertyCharts';

const StyledContainer = styled(Container)(({ theme }) => ({
    marginTop: theme.spacing(4),
}));

const StyledSection = styled(Box)(({ theme }) => ({
    marginBottom: theme.spacing(4),
}));

// Create separate page components
const DashboardPage = () => (
    <>
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
    </>
);

const AnalyticsPage = () => (
    <StyledSection>
        <Typography variant="h4" gutterBottom>
            Property Analysis
        </Typography>
        <PropertyCharts />
    </StyledSection>
);

// Navigation component
const Navigation = () => {
    const location = useLocation();
    const navigate = useNavigate();
    
    const handleChange = (event: React.SyntheticEvent, newValue: string) => {
        navigate(newValue);
    };

    return (
        <Tabs 
            value={location.pathname} 
            onChange={handleChange}
            sx={{ backgroundColor: 'white', borderBottom: 1, borderColor: 'divider' }}
        >
            <Tab label="Dashboard" value="/" />
            <Tab label="Analytics" value="/analytics" />
        </Tabs>
    );
};

function App() {
    return (
        <Router>
            <Box sx={{ flexGrow: 1 }}>
                <AppBar position="static">
                    <Toolbar>
                        <Typography variant="h6" sx={{ flexGrow: 1 }}>
                            FundaMental - Amsterdam Property Analysis
                        </Typography>
                    </Toolbar>
                </AppBar>
                
                <Navigation />

                <StyledContainer>
                    <Routes>
                        <Route path="/" element={<DashboardPage />} />
                        <Route path="/analytics" element={<AnalyticsPage />} />
                    </Routes>
                </StyledContainer>
            </Box>
        </Router>
    );
}

export default App; 