import React, { useEffect, useState } from 'react';
import { PropertyStats as Stats } from '../types/property';
import { api } from '../services/api';
import { Card, CardContent, Typography, Grid } from '@material-ui/core';
import { makeStyles } from '@material-ui/core/styles';

const useStyles = makeStyles({
    root: {
        minWidth: 275,
        margin: '20px 0',
    },
    title: {
        fontSize: 14,
        color: '#666',
    },
    value: {
        marginBottom: 12,
        fontSize: 24,
    },
});

const PropertyStats: React.FC = () => {
    const classes = useStyles();
    const [stats, setStats] = useState<Stats | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchStats = async () => {
            try {
                const data = await api.getPropertyStats();
                setStats(data);
            } catch (error) {
                console.error('Failed to fetch stats:', error);
            } finally {
                setLoading(false);
            }
        };

        fetchStats();
    }, []);

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
                <Card className={classes.root}>
                    <CardContent>
                        <Typography className={classes.title} color="textSecondary" gutterBottom>
                            Total Properties
                        </Typography>
                        <Typography className={classes.value} variant="h5" component="h2">
                            {stats.total_properties}
                        </Typography>
                    </CardContent>
                </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={4}>
                <Card className={classes.root}>
                    <CardContent>
                        <Typography className={classes.title} color="textSecondary" gutterBottom>
                            Average Price
                        </Typography>
                        <Typography className={classes.value} variant="h5" component="h2">
                            {formatPrice(stats.average_price)}
                        </Typography>
                    </CardContent>
                </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={4}>
                <Card className={classes.root}>
                    <CardContent>
                        <Typography className={classes.title} color="textSecondary" gutterBottom>
                            Average Price per m²
                        </Typography>
                        <Typography className={classes.value} variant="h5" component="h2">
                            {formatPrice(stats.price_per_sqm)}/m²
                        </Typography>
                    </CardContent>
                </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={4}>
                <Card className={classes.root}>
                    <CardContent>
                        <Typography className={classes.title} color="textSecondary" gutterBottom>
                            Total Sold
                        </Typography>
                        <Typography className={classes.value} variant="h5" component="h2">
                            {stats.total_sold}
                        </Typography>
                    </CardContent>
                </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={4}>
                <Card className={classes.root}>
                    <CardContent>
                        <Typography className={classes.title} color="textSecondary" gutterBottom>
                            Average Days to Sell
                        </Typography>
                        <Typography className={classes.value} variant="h5" component="h2">
                            {stats.avg_days_to_sell.toFixed(1)} days
                        </Typography>
                    </CardContent>
                </Card>
            </Grid>
        </Grid>
    );
};

export default PropertyStats; 