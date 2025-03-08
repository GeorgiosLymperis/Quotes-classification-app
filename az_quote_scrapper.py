from base_scraper import BaseScraper

class AzQuoteScraper(BaseScraper):
    def scrape(self, url):
        page = self.get_page(url)
        soup = self.parse_page(page)
        quotes = []
        for block in soup.find_all('div', class_='wrap-block'):
            quotes.append({
                'quote': self.get_quote(block),
                'author': self.get_author(block),
                'tags': self.get_tags(block),
                'likes': self.get_likes(block)
            })
        return quotes
    @staticmethod
    def get_quote(block):
        return block.find('a', class_='title').get_text()
    @staticmethod
    def get_author(block):
        author_tag = block.find('div', class_='author').get_text(strip=True)
        return author_tag if author_tag else 'Unknown'
    @staticmethod
    def get_tags(block):
        tags = []
        tags_block = block.find('div', class_='mytags')
        for tag in tags_block.find_all('a'):
            tags.append(tag.get_text())

        return tags
    @staticmethod
    def get_likes(block):
        icons = block.find('div', class_='share-icons')
        return icons.find('a', class_='heart24 heart24-off').get_text()

if __name__ == '__main__':
    url = 'https://www.azquotes.com/quotes/topics/inspirational.html'
    scraper = AzQuoteScraper()
    quotes = scraper.scrape(url)
    print(quotes)