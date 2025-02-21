import React, { useState, createContext, useContext, useEffect } from 'react';
import { Container, Typography, AppBar, Toolbar, Box, Tabs, Tab, Paper, Stack, Alert } from '@mui/material';
import { styled } from '@mui/material/styles';
import { BrowserRouter as Router, Routes, Route, useLocation, useNavigate, Navigate } from 'react-router-dom';
import { LocalizationProvider } from '@mui/x-date-pickers';
import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs';
import { DatePicker } from '@mui/x-date-pickers/DatePicker';
import dayjs, { Dayjs } from 'dayjs';
import { api } from './services/api';
import PropertyMap from './components/PropertyMap';
import PropertyStats from './components/PropertyStats';
import PropertyCharts from './components/PropertyCharts';
import MetropolitanAreaList from './components/MetropolitanAreaList';
import MetropolitanAreaSelector from './components/MetropolitanAreaSelector';
import TelegramConfig from './components/TelegramConfig';

// Create Metropolitan Context
interface MetropolitanContextType {
    selectedMetroArea: number | null;
    setSelectedMetroArea: (id: number | null) => void;
}

const MetropolitanContext = createContext<MetropolitanContextType | undefined>(undefined);

export const useMetropolitanArea = () => {
    const context = useContext(MetropolitanContext);
    if (context === undefined) {
        throw new Error('useMetropolitanArea must be used within a MetropolitanProvider');
    }
    return context;
};

const StyledContainer = styled(Container)(({ theme }) => ({
    marginTop: theme.spacing(4),
}));

const StyledSection = styled(Box)(({ theme }) => ({
    marginBottom: theme.spacing(4),
}));

// Add a new component to handle setup check and redirection
const SetupCheck = ({ children }: { children: React.ReactNode }) => {
    const [needsSetup, setNeedsSetup] = useState<boolean | null>(null);
    const [error, setError] = useState<string | null>(null);
    const location = useLocation();
    const navigate = useNavigate();

    useEffect(() => {
        const checkSetup = async () => {
            try {
                const result = await api.checkInitialSetup();
                setNeedsSetup(result.needs_setup);
                
                // Redirect to config if setup is needed and we're not already there
                if (result.needs_setup && location.pathname !== '/config') {
                    navigate('/config');
                }
            } catch (err) {
                setError('Failed to check application setup status');
                console.error('Setup check failed:', err);
            }
        };

        checkSetup();
    }, [navigate, location.pathname]);

    if (error) {
        return <Alert severity="error">{error}</Alert>;
    }

    if (needsSetup === null) {
        return null; // or a loading spinner
    }

    return <>{children}</>;
};

// Create separate page components
const DashboardPage = () => {
    const [startDate, setStartDate] = useState<Dayjs | null>(dayjs().subtract(1, 'year'));
    const [endDate, setEndDate] = useState<Dayjs | null>(dayjs());
    const { selectedMetroArea } = useMetropolitanArea();

    const dateRange = {
        startDate: startDate?.format('YYYY-MM-DD'),
        endDate: endDate?.format('YYYY-MM-DD')
    };

    return (
        <>
            <StyledSection>
                <Paper sx={{ p: 2, mb: 3 }}>
                    <Typography variant="h6" gutterBottom>
                        Date Range
                    </Typography>
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
    const { selectedMetroArea } = useMetropolitanArea();

    return (
        <>
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
    <>
        <StyledSection>
            <Typography variant="h4" gutterBottom>
                Metropolitan Area Configuration
            </Typography>
            <MetropolitanAreaList />
        </StyledSection>

        <StyledSection>
            <Typography variant="h4" gutterBottom>
                Notifications
            </Typography>
            <TelegramConfig />
        </StyledSection>
    </>
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
    const [selectedMetroArea, setSelectedMetroArea] = useState<number | null>(null);

    return (
        <Router>
            <LocalizationProvider dateAdapter={AdapterDayjs}>
                <MetropolitanContext.Provider value={{ selectedMetroArea, setSelectedMetroArea }}>
                    <Box sx={{ flexGrow: 1 }}>
                        <AppBar position="static">
                            <Toolbar sx={{ display: 'flex', justifyContent: 'space-between' }}>
                                <Typography variant="h6">
                                    FundaMental - Property Analysis
                                </Typography>
                                <Box sx={{ width: 300 }}>
                                    <MetropolitanAreaSelector
                                        value={selectedMetroArea}
                                        onChange={setSelectedMetroArea}
                                    />
                                </Box>
                            </Toolbar>
                        </AppBar>
                        
                        <Navigation />

                        <StyledContainer>
                            <SetupCheck>
                                <Routes>
                                    <Route path="/" element={<DashboardPage />} />
                                    <Route path="/analytics" element={<AnalyticsPage />} />
                                    <Route path="/config" element={<ConfigPage />} />
                                </Routes>
                            </SetupCheck>
                        </StyledContainer>
                    </Box>
                </MetropolitanContext.Provider>
            </LocalizationProvider>
        </Router>
    );
}

export default App; 