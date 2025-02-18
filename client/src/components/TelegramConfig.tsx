import React, { useState, useEffect } from 'react';
import {
    Paper,
    Typography,
    TextField,
    Switch,
    FormControlLabel,
    Button,
    Box,
    Alert,
    CircularProgress,
    Link,
    Divider
} from '@mui/material';
import { api } from '../services/api';

interface TelegramConfigData {
    bot_token: string;
    chat_id: string;
    is_enabled: boolean;
}

const TelegramConfig: React.FC = () => {
    const [config, setConfig] = useState<TelegramConfigData>({
        bot_token: '',
        chat_id: '',
        is_enabled: false,
    });
    const [originalToken, setOriginalToken] = useState<string>('');
    const [hasTokenChanged, setHasTokenChanged] = useState(false);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [testing, setTesting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);

    useEffect(() => {
        const fetchConfig = async () => {
            try {
                const data = await api.getTelegramConfig();
                setConfig(data);
                setOriginalToken(data.bot_token);
                setLoading(false);
            } catch (err) {
                setError('Failed to load Telegram configuration');
                setLoading(false);
            }
        };
        fetchConfig();
    }, []);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setSaving(true);
        setError(null);
        setSuccess(null);

        try {
            // Only send the bot token if it has been changed
            const updateData = {
                ...config,
                bot_token: hasTokenChanged ? config.bot_token : originalToken
            };
            await api.updateTelegramConfig(updateData);
            setSuccess('Telegram configuration updated successfully');
            
            // After successful update, refresh the config
            const newData = await api.getTelegramConfig();
            setConfig(newData);
            setOriginalToken(newData.bot_token);
            setHasTokenChanged(false);
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to update Telegram configuration');
        } finally {
            setSaving(false);
        }
    };

    const handleTestNotification = async () => {
        setTesting(true);
        setError(null);
        setSuccess(null);

        try {
            await api.testTelegramConfig();
            setSuccess('Test notification sent successfully! Check your Telegram.');
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to send test notification');
        } finally {
            setTesting(false);
        }
    };

    const handleTokenChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const newToken = e.target.value;
        setConfig({ ...config, bot_token: newToken });
        setHasTokenChanged(true);
    };

    if (loading) {
        return <CircularProgress />;
    }

    const isConfigured = Boolean(originalToken || (hasTokenChanged && config.bot_token)) && config.chat_id;

    return (
        <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
                Telegram Notifications
            </Typography>
            
            <Typography variant="body2" color="text.secondary" gutterBottom>
                Configure Telegram notifications for new properties. Follow these steps:
            </Typography>
            <Box sx={{ mb: 2 }}>
                <ol>
                    <li>
                        Create a new bot using{' '}
                        <Link href="https://t.me/BotFather" target="_blank" rel="noopener">
                            @BotFather
                        </Link>
                    </li>
                    <li>Copy the bot token provided by BotFather</li>
                    <li>
                        Start a chat with your bot or add it to a group where you want to receive notifications
                    </li>
                    <li>
                        Get your chat ID using{' '}
                        <Link href="https://t.me/userinfobot" target="_blank" rel="noopener">
                            @userinfobot
                        </Link>
                    </li>
                </ol>
            </Box>

            <form onSubmit={handleSubmit}>
                <TextField
                    fullWidth
                    label="Bot Token"
                    value={config.bot_token}
                    onChange={handleTokenChange}
                    margin="normal"
                    required
                    type="password"
                    placeholder={hasTokenChanged ? '' : 'Enter new token or leave unchanged'}
                />

                <TextField
                    fullWidth
                    label="Chat ID"
                    value={config.chat_id}
                    onChange={(e) => setConfig({ ...config, chat_id: e.target.value })}
                    margin="normal"
                    required
                    helperText="Your Telegram user ID or group chat ID"
                />

                <Box sx={{ mt: 2, mb: 2 }}>
                    <FormControlLabel
                        control={
                            <Switch
                                checked={config.is_enabled}
                                onChange={(e) => setConfig({ ...config, is_enabled: e.target.checked })}
                            />
                        }
                        label="Enable Notifications"
                    />
                </Box>

                {error && (
                    <Alert severity="error" sx={{ mb: 2 }}>
                        {error}
                    </Alert>
                )}

                {success && (
                    <Alert severity="success" sx={{ mb: 2 }}>
                        {success}
                    </Alert>
                )}

                <Box sx={{ display: 'flex', gap: 2 }}>
                    <Button
                        variant="contained"
                        color="primary"
                        type="submit"
                        disabled={saving}
                    >
                        {saving ? <CircularProgress size={24} /> : 'Save Configuration'}
                    </Button>

                    <Button
                        variant="outlined"
                        color="secondary"
                        onClick={handleTestNotification}
                        disabled={testing || !isConfigured}
                    >
                        {testing ? <CircularProgress size={24} /> : 'Send Test Notification'}
                    </Button>
                </Box>
            </form>
        </Paper>
    );
};

export default TelegramConfig; 