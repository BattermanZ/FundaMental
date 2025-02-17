import React, { useEffect, useState } from 'react';
import { Property } from '../types/property';
import { api } from '../services/api';
import {
    Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend,
    BarChart, Bar, ScatterChart, Scatter, ResponsiveContainer,
    ComposedChart, Area, Label
} from 'recharts';
import { Box, Typography, CircularProgress, Paper, Grid } from '@mui/material';
import * as d3 from 'd3';
import PriceHeatmap from './PriceHeatmap';

const COLORS = [
    '#1f77b4', '#ff7f0e', '#2ca02c', '#d62728', '#9467bd',
    '#8c564b', '#e377c2', '#7f7f7f', '#bcbd22', '#17becf'
];

interface PriceByPostalCodeData {
    postal_code: string;
    avg_price: number;
    median_price: number;
    count: number;
}

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

    // Prepare data for Price vs Living Area Scatter Plot
    const preparePriceVsAreaData = () => {
        return properties
            .filter(p => p.living_area && p.price)
            .map(p => ({
                living_area: p.living_area,
                price: p.price,
                postal_code: p.postal_code.substring(0, 4),
                num_rooms: p.num_rooms || 1,
                price_per_sqm: p.price / p.living_area
            }));
    };

    // Replace prepareBoxPlotData with new preparePriceByPostalCodeData
    const preparePriceByPostalCodeData = () => {
        const postalGroups = d3.group(
            properties.filter(p => p.price),
            d => d.postal_code.substring(0, 4)
        );

        return Array.from(postalGroups, ([postal_code, group]) => ({
            postal_code,
            avg_price: d3.mean(group, d => d.price) || 0,
            median_price: d3.median(group, d => d.price) || 0,
            count: group.length
        })).sort((a, b) => b.avg_price - a.avg_price);
    };

    // Prepare Time Series Data
    const prepareTimeSeriesData = () => {
        const soldProperties = properties
            .filter(p => p.status === 'sold' && p.selling_date)
            .sort((a, b) => new Date(a.selling_date).getTime() - new Date(b.selling_date).getTime());

        // Group by month
        const monthlyData = d3.group(soldProperties, d => 
            new Date(d.selling_date).toISOString().substring(0, 7)
        );

        return Array.from(monthlyData, ([month, group]) => ({
            month,
            avg_price: d3.mean(group, d => d.price) || 0,
            median_price: d3.median(group, d => d.price) || 0,
            avg_days_to_sell: d3.mean(group, d => {
                if (!d.listing_date || !d.selling_date) return null;
                return (new Date(d.selling_date).getTime() - new Date(d.listing_date).getTime()) / (1000 * 60 * 60 * 24);
            }) || 0,
            count: group.length
        }));
    };

    // Prepare Price per Square Meter Analysis
    const preparePricePerSqmData = () => {
        const postalGroups = d3.group(
            properties.filter(p => p.living_area && p.price),
            d => d.postal_code.substring(0, 4)
        );

        return Array.from(postalGroups, ([postal_code, group]) => ({
            postal_code,
            avg_price_per_sqm: d3.mean(group, d => d.price / d.living_area) || 0,
            median_price_per_sqm: d3.median(group, d => d.price / d.living_area) || 0,
            count: group.length
        })).sort((a, b) => b.avg_price_per_sqm - a.avg_price_per_sqm);
    };

    // Calculate regression line for scatter plot
    const calculateRegressionLine = (data: any[]) => {
        const xValues = data.map(d => d.living_area);
        const yValues = data.map(d => d.price);
        
        const xMean = d3.mean(xValues) || 0;
        const yMean = d3.mean(yValues) || 0;
        
        const ssXX = d3.sum(xValues, x => Math.pow(x - xMean, 2));
        const ssXY = d3.sum(data, d => (d.living_area - xMean) * (d.price - yMean));
        
        const slope = ssXY / ssXX;
        const intercept = yMean - slope * xMean;
        
        const minX = Math.min(...xValues);
        const maxX = Math.max(...xValues);
        
        return [
            { x: minX, y: slope * minX + intercept },
            { x: maxX, y: slope * maxX + intercept }
        ];
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

    const scatterData = preparePriceVsAreaData();
    const priceByPostalCodeData = preparePriceByPostalCodeData();
    const timeSeriesData = prepareTimeSeriesData();
    const pricePerSqmData = preparePricePerSqmData();
    const regressionLine = calculateRegressionLine(scatterData);

    return (
        <Box mt={4}>
            <Grid container spacing={3}>
                {/* Price vs Living Area Scatter Plot */}
                <Grid item xs={12}>
                    <Paper sx={{ p: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            Price vs Living Area
                        </Typography>
                        <ResponsiveContainer width="100%" height={400}>
                            <ScatterChart margin={{ top: 20, right: 20, bottom: 20, left: 60 }}>
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis 
                                    dataKey="living_area" 
                                    name="Living Area" 
                                    unit="m²"
                                    type="number"
                                >
                                    <Label value="Living Area (m²)" offset={-10} position="insideBottom" />
                                </XAxis>
                                <YAxis 
                                    dataKey="price" 
                                    name="Price" 
                                    unit="€"
                                    tickFormatter={(value) => `€${(value/1000)}k`}
                                >
                                    <Label value="Price (€)" angle={-90} position="insideLeft" offset={10} />
                                </YAxis>
                                <Tooltip 
                                    formatter={(value: any, name: string) => {
                                        if (name === 'Price') return `€${Number(value).toLocaleString()}`;
                                        if (name === 'Living Area') return `${value} m²`;
                                        return value;
                                    }}
                                />
                                <Legend />
                                <Scatter 
                                    name="Properties" 
                                    data={scatterData} 
                                    fill="#8884d8"
                                />
                                <Line
                                    name="Regression Line"
                                    data={regressionLine}
                                    dataKey="y"
                                    stroke="#ff7300"
                                    dot={false}
                                />
                            </ScatterChart>
                        </ResponsiveContainer>
                    </Paper>
                </Grid>

                {/* Price Heatmaps */}
                <Grid item xs={12}>
                    <PriceHeatmap properties={properties} metric="price" />
                </Grid>
                <Grid item xs={12}>
                    <PriceHeatmap properties={properties} metric="price_per_sqm" />
                </Grid>

                {/* Price by Postal Code */}
                <Grid item xs={12}>
                    <Paper sx={{ p: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            Price by Postal Code
                        </Typography>
                        <ResponsiveContainer width="100%" height={400}>
                            <BarChart data={priceByPostalCodeData} margin={{ top: 20, right: 20, bottom: 20, left: 60 }}>
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis dataKey="postal_code" />
                                <YAxis 
                                    tickFormatter={(value) => `€${(value/1000)}k`}
                                >
                                    <Label value="Price (€)" angle={-90} position="insideLeft" offset={10} />
                                </YAxis>
                                <Tooltip 
                                    formatter={(value: any) => `€${Number(value).toLocaleString()}`}
                                />
                                <Legend />
                                <Bar 
                                    dataKey="avg_price" 
                                    fill="#8884d8" 
                                    name="Average Price"
                                />
                                <Bar 
                                    dataKey="median_price" 
                                    fill="#82ca9d" 
                                    name="Median Price"
                                />
                            </BarChart>
                        </ResponsiveContainer>
                    </Paper>
                </Grid>

                {/* Time Series */}
                <Grid item xs={12}>
                    <Paper sx={{ p: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            Price Trends Over Time
                        </Typography>
                        <ResponsiveContainer width="100%" height={400}>
                            <ComposedChart data={timeSeriesData} margin={{ top: 20, right: 20, bottom: 20, left: 60 }}>
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis dataKey="month" />
                                <YAxis 
                                    yAxisId="left"
                                    tickFormatter={(value) => `€${(value/1000)}k`}
                                >
                                    <Label value="Price (€)" angle={-90} position="insideLeft" offset={10} />
                                </YAxis>
                                <YAxis 
                                    yAxisId="right" 
                                    orientation="right"
                                    label={{ value: 'Days to Sell', angle: 90, position: 'insideRight' }}
                                />
                                <Tooltip />
                                <Legend />
                                <Line
                                    yAxisId="left"
                                    type="monotone"
                                    dataKey="avg_price"
                                    stroke="#8884d8"
                                    name="Average Price"
                                />
                                <Line
                                    yAxisId="left"
                                    type="monotone"
                                    dataKey="median_price"
                                    stroke="#82ca9d"
                                    name="Median Price"
                                />
                                <Line
                                    yAxisId="right"
                                    type="monotone"
                                    dataKey="avg_days_to_sell"
                                    stroke="#ffc658"
                                    name="Avg Days to Sell"
                                />
                                <Area
                                    yAxisId="left"
                                    type="monotone"
                                    dataKey="count"
                                    fill="#8884d8"
                                    opacity={0.1}
                                    name="Number of Sales"
                                />
                            </ComposedChart>
                        </ResponsiveContainer>
                    </Paper>
                </Grid>

                {/* Price per Square Meter Analysis */}
                <Grid item xs={12}>
                    <Paper sx={{ p: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            Price per Square Meter by Postal Code
                        </Typography>
                        <ResponsiveContainer width="100%" height={400}>
                            <BarChart data={pricePerSqmData} margin={{ top: 20, right: 20, bottom: 20, left: 60 }}>
                                <CartesianGrid strokeDasharray="3 3" />
                                <XAxis dataKey="postal_code" />
                                <YAxis 
                                    tickFormatter={(value) => `€${value.toFixed(0)}`}
                                >
                                    <Label value="Price per m² (€)" angle={-90} position="insideLeft" offset={10} />
                                </YAxis>
                                <Tooltip 
                                    formatter={(value: any) => `€${Number(value).toFixed(0)}/m²`}
                                />
                                <Legend />
                                <Bar 
                                    dataKey="avg_price_per_sqm" 
                                    fill="#8884d8" 
                                    name="Average Price/m²"
                                />
                                <Bar 
                                    dataKey="median_price_per_sqm" 
                                    fill="#82ca9d" 
                                    name="Median Price/m²"
                                />
                            </BarChart>
                        </ResponsiveContainer>
                    </Paper>
                </Grid>
            </Grid>
        </Box>
    );
};

export default PropertyCharts; 