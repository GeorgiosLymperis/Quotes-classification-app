from base_scraper import BaseScraper

class GoodreadsScraper(BaseScraper):
    def scrape(self, url):
        page = self.get_page(url)
        soup = self.parse_page(page)
        quotes = []

        for block in soup.find_all('div', class_='quote mediumText'):
            quotes.append({
                'quote': self.get_quote(block),
                'author': self.get_author(block),
                'tags': self.get_tags(block),
                'likes': self.get_likes(block)
            })
        return quotes
    @staticmethod
    def get_quote(block):
        return block.find('div', class_='quoteText').get_text(strip=True)
    
    @staticmethod
    def get_author(block):
        author_tag = block.find('span', class_='authorOrTitle').get_text(strip=True)
        return author_tag if author_tag else 'Unknown'

    @staticmethod
    def get_likes(block):
        likes_box = block.find('div', class_='quoteFooter')
        likes = likes_box.find('a', class_='smallText').get_text(strip=True)
        return likes.split(' ')[0]
    
    @staticmethod
    def get_tags(block):
        tags = []
        tags_block = block.find('div', class_='quoteFooter')
        for tag in tags_block.find_all('a'):
            tags.append(tag.get_text())
        return tags[:-1]
    
if __name__ == '__main__':
    url = 'https://www.goodreads.com/quotes/tag/motivational-quotes'
    scraper = GoodreadsScraper()
    quotes = scraper.scrape(url)
    print(quotes)