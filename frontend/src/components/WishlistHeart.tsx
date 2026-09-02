import { useTranslation } from 'react-i18next'
import { useMe, useToggleWishlist, useWishlist } from '../api/hooks'
import { HeartIcon, IconButton } from './ui'
import { cx } from '../lib/cx'

/**
 * The wishlist heart (E8) — ONE component for every screen that draws one
 * (card, product page, wishlist grid), so the on/off logic exists once.
 *
 * Its state derives from the wishlist QUERY, not from local state: the same
 * heart on the shop grid and the product page must agree, and two Sets of
 * ids would drift the moment one screen toggled. aria-pressed makes it a
 * real toggle to a screen reader; the label stays constant ("Wishlist") and
 * the STATE is what changes — the button's name should not flip mid-press.
 *
 * Signed-out visitors see a disabled heart with the reason in its tooltip —
 * consistent with carts requiring login (decision #9), and honest where a
 * heart that silently navigates to /login would be a trap dressed as a
 * button.
 */
export function WishlistHeart({
  productId,
  className,
}: {
  productId: number
  className?: string
}) {
  const { t } = useTranslation()
  const me = useMe()
  const wishlist = useWishlist(!!me.data)
  const toggle = useToggleWishlist()

  const hearted = wishlist.data?.some((p) => p.id === productId) ?? false
  const signedIn = !!me.data

  return (
    <IconButton
      label={t('common:actions.wishlist')}
      tone="bare"
      disabled={!signedIn || toggle.isPending}
      aria-pressed={signedIn ? hearted : undefined}
      title={signedIn ? undefined : t('account:wishlist.signInToSave')}
      onClick={() => toggle.mutate({ productId, hearted: !hearted })}
      className={cx(
        'size-8 bg-panel-soft',
        hearted ? 'text-brand' : 'text-ink-faint',
        className,
      )}
    >
      <HeartIcon filled={hearted} />
    </IconButton>
  )
}
