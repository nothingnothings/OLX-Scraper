#!/usr/bin/env python3

import sys
import random
from seleniumbase import SB


def fetch(url):

    with SB(
        browser="chrome",
        headless=False,
        uc=True
    ) as sb:

        sb.open(url)

        sb.wait_for_element(
            "body",
            timeout=20
        )

        # human-like delay
        sb.sleep(
            random.uniform(3,6)
        )

        html = sb.get_page_source()

        # Cloudflare detection
        if (
            "Just a moment" in html
            or "cf-challenge" in html
            or "Attention Required" in html
        ):
            print(
                "CLOUDFLARE_BLOCK",
                file=sys.stderr
            )
            sys.exit(2)

        print(html)


if __name__ == "__main__":

    url = sys.argv[1]

    fetch(url)