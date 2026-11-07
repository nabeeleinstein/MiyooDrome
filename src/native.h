#ifndef MIYOODROME_NATIVE_H
#define MIYOODROME_NATIVE_H

typedef struct {
    double position;
    double duration;
    int is_playing;
    int is_paused;
    int finished;
} AudioState;

void c_log(const char *msg);
void c_logf(const char *fmt, const char *detail);
void c_logd(const char *fmt, int val);
int pollEvents(void);
int refreshScreenPtr(unsigned char *pixels);
int init(void);
void quit(void);
int audio_init(void);
int audio_load(const char *path);
int audio_load_mem(void *data, int size);
int audio_play(void);
void audio_pause(void);
void audio_resume(void);
void audio_toggle_pause(void);
void audio_stop(void);
int audio_is_playing(void);
int audio_is_paused(void);
double audio_get_position(void);
double audio_get_duration(void);
double audio_get_file_duration(const char *path);
int audio_seek(double position);
void audio_set_volume(int volume);
int audio_check_finished(void);
void audio_get_state(AudioState *state);
void audio_quit(void);

#endif
