import React, { useEffect, useState } from 'react';
import { PropertyStats as Stats, DateRange } from '../types/property';
import { api } from '../services/api';
import { Card, CardContent, Typography, Grid } from '@mui/material';
import { styled } from '@mui/material/styles';

const StyledCard = styled(Card)(({ theme }) => ({
    minWidth: 275,
    margin: '20px 0',
}));

const TitleTypography = styled(Typography)(() => ({
    fontSize: 14,
    color: '#666',
}));

const ValueTypography = styled(Typography)(() => ({
    marginBottom: 12,
    fontSize: 24,
}));

interface PropertyStatsProps {
    dateRange: DateRange;
    metropolitanAreaId?: number | null;
}

const PropertyStats: React.FC<PropertyStatsProps> = ({ dateRange, metropolitanAreaId }) => {
    const [stats, setStats] = useState<Stats | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchStats = async () => {
            try {
                setLoading(true);
                const data = await api.getPropertyStats(dateRange, metropolitanAreaId);
                setStats(data);
            } catch (error) {
                console.error('Failed to fetch stats:', error);
            } finally {
                setLoading(false);
            }
        };

        fetchStats();
    }, [dateRange, metropolitanAreaId]);

    if (loading) {
        return <div>Loading statistics...</div>;
    }

    if (!stats) {
        return <div>No statistics available</div>;
    }

    const formatPrice = (price: number) => 
        `€${price.toLocaleString(undefined, { maximumFractionDigits: 0 })}`;

    return (
        <Grid container spacing={3}>
            <Grid item xs={12} sm={6} md={4}>
                <StyledCard>
                    <CardContent>
                        <TitleTypography color="textSecondary" gutterBottom>
                            Total Properties
                        </TitleTypography>
                        <ValueTypography variant="h5">
                            {stats.total_properties}
                        </ValueTypography>
                    </CardContent>
                </StyledCard>
            </Grid>

            <Grid item xs={12} sm={6} md={4}>
                <StyledCard>
                    <CardContent>
                        <TitleTypography color="textSecondary" gutterBottom>
                            Average Price
                        </TitleTypography>
                        <ValueTypography variant="h5">
                            {formatPrice(stats.average_price)}
                        </ValueTypography>
                    </CardContent>
                </StyledCard>
            </Grid>

            <Grid item xs={12} sm={6} md={4}>
                <StyledCard>
                    <CardContent>
                        <TitleTypography color="textSecondary" gutterBottom>
                            Average Price per m²
                        </TitleTypography>
                        <ValueTypography variant="h5">
                            {formatPrice(stats.price_per_sqm)}/m²
                        </ValueTypography>
                    </CardContent>
                </StyledCard>
            </Grid>

            <Grid item xs={12} sm={6} md={4}>
                <StyledCard>
                    <CardContent>
                        <TitleTypography color="textSecondary" gutterBottom>
                            Total Active
                        </TitleTypography>
                        <ValueTypography variant="h5">
                            {stats.total_active}
                        </ValueTypography>
                    </CardContent>
                </StyledCard>
            </Grid>

            <Grid item xs={12} sm={6} md={4}>
                <StyledCard>
                    <CardContent>
                        <TitleTypography color="textSecondary" gutterBottom>
                            Total Sold
                        </TitleTypography>
                        <ValueTypography variant="h5">
                            {stats.total_sold}
                        </ValueTypography>
                    </CardContent>
                </StyledCard>
            </Grid>

            <Grid item xs={12} sm={6} md={4}>
                <StyledCard>
                    <CardContent>
                        <TitleTypography color="textSecondary" gutterBottom>
                            Average Days to Sell
                        </TitleTypography>
                        <ValueTypography variant="h5">
                            {stats.avg_days_to_sell.toFixed(1)} days
                        </ValueTypography>
                    </CardContent>
                </StyledCard>
            </Grid>
        </Grid>
    );
};

export default PropertyStats; 