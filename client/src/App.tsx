import React, { useState } from 'react';
import { Container, Typography, AppBar, Toolbar, Box, Tabs, Tab, Paper, Stack } from '@mui/material';
import { styled } from '@mui/material/styles';
import { BrowserRouter as Router, Routes, Route, useLocation, useNavigate } from 'react-router-dom';
import { LocalizationProvider } from '@mui/x-date-pickers';
import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs';
import { DatePicker } from '@mui/x-date-pickers/DatePicker';
import dayjs, { Dayjs } from 'dayjs';
import PropertyMap from './components/PropertyMap';
import PropertyStats from './components/PropertyStats';
import PropertyCharts from './components/PropertyCharts';
import MetropolitanAreaList from './components/MetropolitanAreaList';
import MetropolitanAreaSelector from './components/MetropolitanAreaSelector';

const StyledContainer = styled(Container)(({ theme }) => ({
    marginTop: theme.spacing(4),
}));

const StyledSection = styled(Box)(({ theme }) => ({
    marginBottom: theme.spacing(4),
}));

// Create separate page components
const DashboardPage = () => {
    const [startDate, setStartDate] = useState<Dayjs | null>(dayjs().subtract(1, 'year'));
    const [endDate, setEndDate] = useState<Dayjs | null>(dayjs());
    const [selectedMetroArea, setSelectedMetroArea] = useState<number | null>(null);

    const dateRange = {
        startDate: startDate?.format('YYYY-MM-DD'),
        endDate: endDate?.format('YYYY-MM-DD')
    };

    return (
        <>
            <StyledSection>
                <Paper sx={{ p: 2, mb: 3 }}>
                    <Typography variant="h6" gutterBottom>
                        Filters
                    </Typography>
                    <Stack spacing={2}>
                        <Stack direction="row" spacing={2}>
                            <DatePicker
                                label="Start Date"
                                value={startDate}
                                onChange={(newValue) => setStartDate(newValue)}
                                maxDate={endDate || undefined}
                                slotProps={{
                                    textField: { fullWidth: true }
                                }}
                            />
                            <DatePicker
                                label="End Date"
                                value={endDate}
                                onChange={(newValue) => setEndDate(newValue)}
                                minDate={startDate || undefined}
                                slotProps={{
                                    textField: { fullWidth: true }
                                }}
                            />
                        </Stack>
                        <MetropolitanAreaSelector
                            value={selectedMetroArea}
                            onChange={setSelectedMetroArea}
                        />
                    </Stack>
                </Paper>
            </StyledSection>

            <StyledSection>
                <Typography variant="h4" gutterBottom>
                    Property Statistics
                </Typography>
                <PropertyStats dateRange={dateRange} metropolitanAreaId={selectedMetroArea} />
            </StyledSection>

            <StyledSection>
                <Typography variant="h4" gutterBottom>
                    Property Map
                </Typography>
                <PropertyMap dateRange={dateRange} metropolitanAreaId={selectedMetroArea} />
            </StyledSection>
        </>
    );
};

const AnalyticsPage = () => {
    const [selectedMetroArea, setSelectedMetroArea] = useState<number | null>(null);

    return (
        <>
            <StyledSection>
                <Paper sx={{ p: 2, mb: 3 }}>
                    <MetropolitanAreaSelector
                        value={selectedMetroArea}
                        onChange={setSelectedMetroArea}
                    />
                </Paper>
            </StyledSection>
            <StyledSection>
                <Typography variant="h4" gutterBottom>
                    Property Analysis
                </Typography>
                <PropertyCharts metropolitanAreaId={selectedMetroArea} />
            </StyledSection>
        </>
    );
};

const ConfigPage = () => (
    <StyledSection>
        <Typography variant="h4" gutterBottom>
            Metropolitan Area Configuration
        </Typography>
        <MetropolitanAreaList />
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
            <Tab label="Configuration" value="/config" />
        </Tabs>
    );
};

function App() {
    return (
        <Router>
            <LocalizationProvider dateAdapter={AdapterDayjs}>
                <Box sx={{ flexGrow: 1 }}>
                    <AppBar position="static">
                        <Toolbar>
                            <Typography variant="h6" sx={{ flexGrow: 1 }}>
                                FundaMental - Property Analysis
                            </Typography>
                        </Toolbar>
                    </AppBar>
                    
                    <Navigation />

                    <StyledContainer>
                        <Routes>
                            <Route path="/" element={<DashboardPage />} />
                            <Route path="/analytics" element={<AnalyticsPage />} />
                            <Route path="/config" element={<ConfigPage />} />
                        </Routes>
                    </StyledContainer>
                </Box>
            </LocalizationProvider>
        </Router>
    );
}

export default App; 