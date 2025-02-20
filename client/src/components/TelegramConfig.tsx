import React, { useEffect, useState } from 'react';
import { TelegramConfig } from '../types/telegram';
import { getTelegramConfig, updateTelegramConfig, testTelegramConfig } from '../api/telegram';
import { toast } from 'react-hot-toast';
import TelegramFiltersComponent from './TelegramFilters';
import {
    Box,
    Typography,
    Paper,
    Switch,
    FormControlLabel,
    TextField,
    Button,
    Stack,
    FormControl,
    InputLabel,
    OutlinedInput,
    FormHelperText
} from '@mui/material';

export default function TelegramConfiguration() {
    const [config, setConfig] = useState<TelegramConfig>({
        is_enabled: false,
        bot_token: '',
        chat_id: '',
    });
    const [loading, setLoading] = useState(false);
    const [testing, setTesting] = useState(false);

    useEffect(() => {
        loadConfig();
    }, []);

    const loadConfig = async () => {
        try {
            const data = await getTelegramConfig();
            setConfig(data);
        } catch (error) {
            toast.error('Failed to load configuration');
        }
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setLoading(true);
        try {
            await updateTelegramConfig(config);
            toast.success('Configuration updated successfully');
        } catch (error) {
            if (error instanceof Error) {
                toast.error(error.message);
            } else {
                toast.error('Failed to update configuration');
            }
        } finally {
            setLoading(false);
        }
    };

    const handleTest = async () => {
        setTesting(true);
        try {
            await testTelegramConfig();
            toast.success('Test message sent successfully');
        } catch (error) {
            if (error instanceof Error) {
                toast.error(error.message);
            } else {
                toast.error('Failed to send test message');
            }
        } finally {
            setTesting(false);
        }
    };

    return (
        <Box sx={{ maxWidth: '1200px', mx: 'auto', py: 4 }}>
            <Typography variant="h4" gutterBottom>Notifications</Typography>
            
            {/* Bot Configuration */}
            <Box sx={{ mb: 4 }}>
                <Typography variant="h5" gutterBottom>Telegram Bot Configuration</Typography>
                <form onSubmit={handleSubmit}>
                    <Stack spacing={2}>
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={config.is_enabled}
                                    onChange={e => setConfig(prev => ({ ...prev, is_enabled: e.target.checked }))}
                                />
                            }
                            label="Enable Notifications"
                        />

                        <FormControl variant="outlined" fullWidth>
                            <InputLabel htmlFor="bot-token">Bot Token</InputLabel>
                            <OutlinedInput
                                id="bot-token"
                                type="password"
                                value={config.bot_token}
                                onChange={e => setConfig(prev => ({ ...prev, bot_token: e.target.value }))}
                                label="Bot Token"
                                placeholder="Enter your bot token from @BotFather"
                            />
                        </FormControl>

                        <FormControl variant="outlined" fullWidth>
                            <InputLabel htmlFor="chat-id">Chat ID</InputLabel>
                            <OutlinedInput
                                id="chat-id"
                                value={config.chat_id}
                                onChange={e => setConfig(prev => ({ ...prev, chat_id: e.target.value }))}
                                label="Chat ID"
                                placeholder="Enter your chat ID"
                            />
                        </FormControl>

                        <Box sx={{ display: 'flex', gap: 1 }}>
                            <Button
                                onClick={handleTest}
                                disabled={testing || loading || !config.is_enabled}
                                variant="outlined"
                                size="small"
                            >
                                {testing ? 'Sending...' : 'Send Test Message'}
                            </Button>
                            <Button
                                type="submit"
                                disabled={loading}
                                variant="contained"
                                size="small"
                            >
                                {loading ? 'Saving...' : 'Save Configuration'}
                            </Button>
                        </Box>
                    </Stack>
                </form>
            </Box>

            {/* Notification Filters */}
            <TelegramFiltersComponent />
        </Box>
    );
} 