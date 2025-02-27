import React, { memo } from 'react';
import {
    Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend,
    BarChart, Bar, ScatterChart, Scatter, ResponsiveContainer,
    ComposedChart, Area, Label
} from 'recharts';
import { Typography, Paper, Grid } from '@mui/material';
import dayjs from 'dayjs';

interface PropertyChartDataProps {
    scatterData: Array<{
        living_area: number;
        price: number;
        postal_code: string;
        num_rooms: number;
        price_per_sqm: number;
    }>;
    priceByPostalCodeData: Array<{
        postal_code: string;
        avg_price: number;
        median_price: number;
        count: number;
    }>;
    timeSeriesData: Array<{
        month: string;
        avg_price: number;
        median_price: number;
        avg_days_to_sell: number;
        count: number;
        active_count: number;
        sold_count: number;
    }>;
    regressionLine: Array<{ x: number; y: number }>;
    priceByRoomsData: Array<{
        rooms: number;
        avg_price: number;
        median_price: number;
        count: number;
    }>;
    priceByRoomsPremiumData: Array<{
        rooms: number;
        count: number;
        median_price: number;
        pricePremium: number;
        percentageIncrease: number;
    }>;
    pricePerSqmData: Array<{
        postal_code: string;
        avg_price_per_sqm: number;
        median_price_per_sqm: number;
        count: number;
    }>;
}

const PropertyChartData: React.FC<PropertyChartDataProps> = ({
    scatterData,
    priceByPostalCodeData,
    timeSeriesData,
    regressionLine,
    priceByRoomsData,
    priceByRoomsPremiumData,
    pricePerSqmData
}) => {
    return (
        <>
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

            {/* Time Series - Property Count and Days to Sell */}
            <Grid item xs={12}>
                <Paper sx={{ p: 3 }}>
                    <Typography variant="h6" gutterBottom>
                        Property Count and Days to Sell
                    </Typography>
                    <ResponsiveContainer width="100%" height={400}>
                        <ComposedChart data={timeSeriesData} margin={{ top: 20, right: 60, bottom: 20, left: 60 }}>
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis 
                                dataKey="month" 
                                tickFormatter={(value) => {
                                    const [year, month] = value.split('-');
                                    return dayjs(`${year}-${month}-01`).format('MMM YY');
                                }}
                            />
                            <YAxis 
                                yAxisId="left"
                                orientation="left"
                                tickFormatter={(value) => value.toFixed(0)}
                            >
                                <Label value="Number of Properties" angle={-90} position="insideLeft" offset={10} />
                            </YAxis>
                            <YAxis 
                                yAxisId="right"
                                orientation="right"
                                tickFormatter={(value) => value.toFixed(0)}
                            >
                                <Label value="Days to Sell" angle={90} position="insideRight" offset={10} />
                            </YAxis>
                            <Tooltip 
                                formatter={(value: any, name: string) => {
                                    if (name === 'avg_days_to_sell') return [value.toFixed(1), 'Avg Days to Sell'];
                                    if (name === 'active_count') return [value, 'Active Properties'];
                                    if (name === 'sold_count') return [value, 'Sold Properties'];
                                    return [value, name];
                                }}
                                labelFormatter={(label) => dayjs(label).format('MMMM YYYY')}
                            />
                            <Legend />
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
                                dataKey="active_count"
                                fill="#82ca9d"
                                stroke="#82ca9d"
                                opacity={0.1}
                                name="Active Properties"
                                stackId="1"
                            />
                            <Area
                                yAxisId="left"
                                type="monotone"
                                dataKey="sold_count"
                                fill="#8884d8"
                                stroke="#8884d8"
                                opacity={0.1}
                                name="Sold Properties"
                                stackId="1"
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

            {/* Market Velocity Dashboard */}
            <Grid item xs={12}>
                <Paper sx={{ p: 3 }}>
                    <Typography variant="h6" gutterBottom>
                        Market Velocity Analysis
                    </Typography>
                    <ResponsiveContainer width="100%" height={400}>
                        <ComposedChart data={timeSeriesData} margin={{ top: 20, right: 60, bottom: 20, left: 60 }}>
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis 
                                dataKey="month" 
                                tickFormatter={(value) => {
                                    const [year, month] = value.split('-');
                                    return dayjs(`${year}-${month}-01`).format('MMM YY');
                                }}
                            />
                            <YAxis 
                                yAxisId="left"
                                orientation="left"
                                tickFormatter={(value) => value.toFixed(0)}
                            >
                                <Label value="Days to Sell" angle={-90} position="insideLeft" offset={10} />
                            </YAxis>
                            <YAxis 
                                yAxisId="right"
                                orientation="right"
                                tickFormatter={(value) => `€${(value/1000)}k`}
                            >
                                <Label value="Price Range (€)" angle={90} position="insideRight" offset={10} />
                            </YAxis>
                            <Tooltip 
                                formatter={(value: any, name: string) => {
                                    if (name.includes('Price')) return `€${Number(value).toLocaleString()}`;
                                    return value.toFixed(1);
                                }}
                                labelFormatter={(label) => dayjs(label).format('MMMM YYYY')}
                            />
                            <Legend />
                            <Bar
                                yAxisId="left"
                                dataKey="avg_days_to_sell"
                                fill="#8884d8"
                                name="Average Days to Sell"
                            />
                            <Line
                                yAxisId="right"
                                type="monotone"
                                dataKey="avg_price"
                                stroke="#82ca9d"
                                name="Average Price"
                            />
                            <Line
                                yAxisId="right"
                                type="monotone"
                                dataKey="median_price"
                                stroke="#ffc658"
                                name="Median Price"
                            />
                        </ComposedChart>
                    </ResponsiveContainer>
                </Paper>
            </Grid>

            {/* Room Distribution Analysis */}
            <Grid item xs={12} md={6}>
                <Paper sx={{ p: 3 }}>
                    <Typography variant="h6" gutterBottom>
                        Price Premium per Additional Room
                    </Typography>
                    <ResponsiveContainer width="100%" height={400}>
                        <ComposedChart 
                            data={priceByRoomsPremiumData} 
                            margin={{ top: 20, right: 60, bottom: 20, left: 60 }}
                        >
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis 
                                dataKey="rooms"
                                label={{ value: 'Number of Rooms', position: 'insideBottom', offset: -10 }}
                            />
                            <YAxis 
                                yAxisId="left"
                                tickFormatter={(value) => `€${(value/1000)}k`}
                            >
                                <Label value="Price Premium (€)" angle={-90} position="center" offset={0} dx={-50} />
                            </YAxis>
                            <YAxis 
                                yAxisId="right"
                                orientation="right"
                                tickFormatter={(value) => `${value.toFixed(1)}%`}
                            >
                                <Label value="Percentage Increase" angle={90} position="center" offset={0} dx={50} />
                            </YAxis>
                            <Tooltip 
                                formatter={(value: any, name: string, props: any) => {
                                    if (name === "Price Premium") {
                                        const roundedValue = Math.round(value / 1000) * 1000;
                                        return [`€${Number(roundedValue).toLocaleString()} (${props.payload.count} properties)`, name];
                                    }
                                    if (name === "Percentage Increase") {
                                        return [`${value.toFixed(1)}% (${props.payload.count} properties)`, name];
                                    }
                                    return [value, name];
                                }}
                            />
                            <Legend 
                                verticalAlign="bottom"
                                align="center"
                                layout="horizontal"
                                wrapperStyle={{
                                    paddingTop: "20px"
                                }}
                            />
                            <Bar 
                                yAxisId="left"
                                dataKey="pricePremium" 
                                fill="#8884d8" 
                                name="Price Premium"
                            />
                            <Line 
                                yAxisId="right"
                                type="monotone"
                                dataKey="percentageIncrease" 
                                stroke="#82ca9d" 
                                name="Percentage Increase"
                            />
                        </ComposedChart>
                    </ResponsiveContainer>
                </Paper>
            </Grid>

            {/* Property Features Impact */}
            <Grid item xs={12} md={6}>
                <Paper sx={{ p: 3 }}>
                    <Typography variant="h6" gutterBottom>
                        Price Impact by Number of Rooms
                    </Typography>
                    <ResponsiveContainer width="100%" height={400}>
                        <BarChart 
                            data={priceByRoomsData.filter(d => d.rooms <= 10)} 
                            margin={{ top: 20, right: 30, bottom: 20, left: 60 }}
                        >
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis 
                                dataKey="rooms"
                                label={{ value: 'Number of Rooms', position: 'insideBottom', offset: -10 }}
                            />
                            <YAxis 
                                tickFormatter={(value) => `€${(value/1000)}k`}
                            >
                                <Label value="Average Price (€)" angle={-90} position="center" dx={-60} />
                            </YAxis>
                            <Tooltip 
                                formatter={(value: any, name: string, props: any) => {
                                    const roundedValue = Math.round(value / 1000) * 1000;
                                    return [`€${Number(roundedValue).toLocaleString()} (${props.payload.count} properties)`, name];
                                }}
                            />
                            <Legend 
                                verticalAlign="bottom"
                                align="center"
                                layout="horizontal"
                                wrapperStyle={{
                                    paddingTop: "20px"
                                }}
                            />
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
        </>
    );
};

export default memo(PropertyChartData); 