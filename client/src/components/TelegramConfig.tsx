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
    FormHelperText,
    Container,
    Divider,
    IconButton,
    Tooltip
} from '@mui/material';
import TelegramIcon from '@mui/icons-material/Telegram';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import SendIcon from '@mui/icons-material/Send';
import SaveIcon from '@mui/icons-material/Save';

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

    const handleEnableToggle = async (enabled: boolean) => {
        try {
            await updateTelegramConfig({
                ...config,
                is_enabled: enabled
            });
            setConfig(prev => ({ ...prev, is_enabled: enabled }));
            toast.success(enabled ? 'Notifications enabled' : 'Notifications disabled');
        } catch (error) {
            toast.error('Failed to update notification status');
            // Revert the switch if the update failed
            setConfig(prev => ({ ...prev, is_enabled: !enabled }));
        }
    };

    const handleSaveConfig = async (e: React.FormEvent) => {
        e.preventDefault();
        setLoading(true);
        try {
            // Only update bot token and chat ID
            await updateTelegramConfig({
                ...config,
                bot_token: config.bot_token,
                chat_id: config.chat_id
            });
            toast.success('Bot configuration updated successfully');
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
        <Container maxWidth="lg" sx={{ py: 4 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 4 }}>
                <TelegramIcon sx={{ fontSize: 32, color: 'primary.main', mr: 2 }} />
                <Typography variant="h4" component="h1">
                    Notifications
                </Typography>
            </Box>

            {/* Bot Configuration */}
            <Paper elevation={2} sx={{ p: 4, mb: 4 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 3 }}>
                    <Typography variant="h5" gutterBottom sx={{ display: 'flex', alignItems: 'center' }}>
                        Telegram Bot Configuration
                        <Tooltip title="Configure your Telegram bot to receive property notifications" arrow>
                            <IconButton size="small" sx={{ ml: 1 }}>
                                <InfoOutlinedIcon fontSize="small" />
                            </IconButton>
                        </Tooltip>
                    </Typography>
                    <FormControlLabel
                        control={
                            <Switch
                                checked={config.is_enabled}
                                onChange={e => handleEnableToggle(e.target.checked)}
                                color="primary"
                            />
                        }
                        label={config.is_enabled ? "Notifications Enabled" : "Notifications Disabled"}
                    />
                </Box>

                <form onSubmit={handleSaveConfig}>
                    <Stack spacing={3}>
                        <FormControl variant="outlined">
                            <InputLabel htmlFor="bot-token">Bot Token</InputLabel>
                            <OutlinedInput
                                id="bot-token"
                                type="password"
                                value={config.bot_token}
                                onChange={e => setConfig(prev => ({ ...prev, bot_token: e.target.value }))}
                                label="Bot Token"
                                placeholder="Enter your bot token from @BotFather"
                                fullWidth
                            />
                            <FormHelperText>
                                Get your bot token from <a href="https://t.me/botfather" target="_blank" rel="noopener noreferrer" style={{ color: '#1976d2' }}>@BotFather</a>
                            </FormHelperText>
                        </FormControl>

                        <FormControl variant="outlined">
                            <InputLabel htmlFor="chat-id">Chat ID</InputLabel>
                            <OutlinedInput
                                id="chat-id"
                                value={config.chat_id}
                                onChange={e => setConfig(prev => ({ ...prev, chat_id: e.target.value }))}
                                label="Chat ID"
                                placeholder="Enter your chat ID"
                                fullWidth
                            />
                            <FormHelperText>
                                Your Telegram chat ID (e.g., -123456789)
                            </FormHelperText>
                        </FormControl>

                        <Box sx={{ display: 'flex', gap: 2, justifyContent: 'flex-end', mt: 2 }}>
                            <Button
                                onClick={handleTest}
                                disabled={testing || loading || !config.is_enabled || !config.bot_token || !config.chat_id}
                                variant="outlined"
                                startIcon={<SendIcon />}
                            >
                                {testing ? 'Sending...' : 'Send Test Message'}
                            </Button>
                            <Button
                                type="submit"
                                disabled={loading}
                                variant="contained"
                                startIcon={<SaveIcon />}
                            >
                                {loading ? 'Saving...' : 'Save Bot Configuration'}
                            </Button>
                        </Box>
                    </Stack>
                </form>
            </Paper>

            {/* Notification Filters */}
            <TelegramFiltersComponent />
        </Container>
    );
} 