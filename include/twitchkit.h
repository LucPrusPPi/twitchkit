#ifndef TWITCHKIT_H
#define TWITCHKIT_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>

/* Opaque client handle (uintptr). */
typedef uintptr_t twitchkit_handle;

/* Create / destroy a client bound to an OAuth access token. */
twitchkit_handle TWITCHKIT_Create(const char *token);
void TWITCHKIT_Destroy(twitchkit_handle handle);

/* Validate token. Returns heap JSON {"login","user_id"} or NULL. Free with TWITCHKIT_Free. */
char *TWITCHKIT_Validate(twitchkit_handle handle);

/* Claim a drop instance. Returns 1 on success, 0 on failure. */
int TWITCHKIT_ClaimDrop(twitchkit_handle handle, const char *drop_instance_id, const char *user_id);

/* Send one minute-watched event. Returns 1 on success, 0 on failure. */
int TWITCHKIT_SendWatch(
    twitchkit_handle handle,
    const char *channel_login,
    const char *channel_id,
    const char *broadcast_id,
    const char *user_id,
    const char *game_name,
    const char *game_id
);

/* Strip optional oauth: prefix. Free with TWITCHKIT_Free. */
char *TWITCHKIT_NormalizeToken(const char *token);

/* Free strings returned by this API. */
void TWITCHKIT_Free(char *p);

#ifdef __cplusplus
}
#endif

#endif /* TWITCHKIT_H */
