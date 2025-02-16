import React, { useEffect, useState } from 'react';
import { Property } from '../types/property';
import { api } from '../services/api';
import {
    LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend,
    BarChart, Bar, ScatterChart, Scatter, ResponsiveContainer,
    PieChart, Pie, Cell
} from 'recharts';
import { Box, Typography, CircularProgress, Paper, Grid } from '@mui/material';

const COLORS = ['#8884d8', '#82ca9d', '#ffc658', '#ff8042', '#a4de6c'];

const PropertyCharts: React.FC = () => {
    const [properties, setProperties] = useState<Property[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetchData = async () => {
            try {
                setLoading(true);
                const defaultDateRange = {
                    startDate: undefined,
                    endDate: undefined
                };
                const data = await api.getAllProperties(defaultDateRange);
                setProperties(data);
                setError(null);
            } catch (error) {
                console.error('Failed to fetch properties:', error);
                setError('Failed to load property data');
            } finally {
                setLoading(false);
            }
        };

        fetchData();
    }, []);

    const calculateMovingAverage = (data: any[], window: number) => {
        return data.map((item, index) => {
            const start = Math.max(0, index - window + 1);
            const windowSlice = data.slice(start, index + 1);
            const average = windowSlice.reduce((sum, curr) => sum + curr.price, 0) / windowSlice.length;
            return {
                ...item,
                movingAverage: Math.round(average)
            };
        });
    };

    const preparePriceHistoryData = () => {
        const soldProperties = properties
            .filter(p => p.status === 'sold' && p.selling_date)
            .sort((a, b) => new Date(a.selling_date).getTime() - new Date(b.selling_date).getTime());

        const baseData = soldProperties.map(p => ({
            date: new Date(p.selling_date).toLocaleDateString(),
            price: p.price,
            pricePerSqm: p.living_area ? Math.round(p.price / p.living_area) : null
        }));

        return calculateMovingAverage(baseData, 5);
    };

    const prepareSizeDistributionData = () => {
        const sizeRanges = Array.from({ length: 10 }, (_, i) => ({
            range: `${i * 20}-${(i + 1) * 20}`,
            count: 0
        }));

        properties.forEach(p => {
            if (p.living_area) {
                const rangeIndex = Math.min(Math.floor(p.living_area / 20), 9);
                sizeRanges[rangeIndex].count++;
            }
        });

        return sizeRanges;
    };

    const preparePriceVsAreaData = () => {
        return properties
            .filter(p => p.living_area && p.price)
            .map(p => ({
                area: p.living_area,
                price: p.price,
                year: p.year_built
            }));
    };

    const preparePropertyTypeData = () => {
        const typeCount: { [key: string]: number } = {};
        properties.forEach(p => {
            const type = p.property_type || 'Unknown';
            typeCount[type] = (typeCount[type] || 0) + 1;
        });

        return Object.entries(typeCount)
            .map(([name, value]) => ({ name, value }))
            .sort((a, b) => b.value - a.value);
    };

    if (loading) {
        return (
            <Box display="flex" justifyContent="center" alignItems="center" height="400px">
                <CircularProgress />
            </Box>
        );
    }

    if (error) {
        return (
            <Box display="flex" justifyContent="center" alignItems="center" height="400px">
                <Typography color="error">{error}</Typography>
            </Box>
        );
    }

    return (
        <Box mt={4}>
            <Grid container spacing={3}>
                <Grid item xs={12}>
                    <Paper style={{ padding: 16 }}>
                        <Typography variant="h6" gutterBottom>
                            Price History
                        </Typography>
                        <ResponsiveContainer width="100%" height={300}>
                            <LineChart data={preparePriceHistoryData()}>
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis dataKey="date" />
                                <YAxis yAxisId="left" />
                                <YAxis yAxisId="right" orientation="right" />
                                <Tooltip />
                                <Legend />
                                <Line
                                    yAxisId="left"
                                    type="monotone"
                                    dataKey="price"
                                    stroke="#8884d8"
                                    name="Price (€)"
                                />
                                <Line
                                    yAxisId="left"
                                    type="monotone"
                                    dataKey="movingAverage"
                                    stroke="#ff8042"
                                    name="5-Day Moving Average (€)"
                                    dot={false}
                                />
                                <Line
                                    yAxisId="right"
                                    type="monotone"
                                    dataKey="pricePerSqm"
                                    stroke="#82ca9d"
                                    name="Price per m² (€)"
                                />
                            </LineChart>
                        </ResponsiveContainer>
                    </Paper>
                </Grid>

                <Grid item xs={12} md={4}>
                    <Paper style={{ padding: 16 }}>
                        <Typography variant="h6" gutterBottom>
                            Property Types
                        </Typography>
                        <ResponsiveContainer width="100%" height={300}>
                            <PieChart>
                                <Pie
                                    data={preparePropertyTypeData()}
                                    dataKey="value"
                                    nameKey="name"
                                    cx="50%"
                                    cy="50%"
                                    outerRadius={80}
                                    label={(entry) => `${entry.name}: ${entry.value}`}
                                >
                                    {preparePropertyTypeData().map((entry, index) => (
                                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                                    ))}
                                </Pie>
                                <Tooltip />
                                <Legend />
                            </PieChart>
                        </ResponsiveContainer>
                    </Paper>
                </Grid>

                <Grid item xs={12} md={4}>
                    <Paper style={{ padding: 16 }}>
                        <Typography variant="h6" gutterBottom>
                            Size Distribution
                        </Typography>
                        <ResponsiveContainer width="100%" height={300}>
                            <BarChart data={prepareSizeDistributionData()}>
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis dataKey="range" />
                                <YAxis />
                                <Tooltip />
                                <Legend />
                                <Bar dataKey="count" fill="#8884d8" name="Number of Properties" />
                            </BarChart>
                        </ResponsiveContainer>
                    </Paper>
                </Grid>

                <Grid item xs={12} md={4}>
                    <Paper style={{ padding: 16 }}>
                        <Typography variant="h6" gutterBottom>
                            Price vs Area
                        </Typography>
                        <ResponsiveContainer width="100%" height={300}>
                            <ScatterChart>
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis dataKey="area" name="Living Area (m²)" />
                                <YAxis dataKey="price" name="Price (€)" />
                                <Tooltip cursor={{ strokeDasharray: '3 3' }} />
                                <Legend />
                                <Scatter
                                    name="Properties"
                                    data={preparePriceVsAreaData()}
                                    fill="#8884d8"
                                />
                            </ScatterChart>
                        </ResponsiveContainer>
                    </Paper>
                </Grid>
            </Grid>
        </Box>
    );
};

export default PropertyCharts; 