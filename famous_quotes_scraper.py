from base_scraper import BaseScraper
from urllib.parse import urljoin

class FamousQuotesScraper(BaseScraper):
    def scrape_page(self, url):
        page = self.get_page(url)
        soup = self.parse_page(page)
        table = soup.find('td', style='padding-left:16px; padding-right:16px;', valign='top')
        quotes = self.get_quote(table)
        authors = self.get_author(table)
        likes = None
        tag = self.get_tags(table)
        assert len(quotes) == len(authors)
        return [{
            'quote': quote,
            'author': author,
            'tags': tag,
            'likes': likes
        } for quote, author in zip(quotes, authors)]

    @staticmethod
    def get_quote(table):
        quotes = []
        for quote in table.find_all('div', style='font-size:12px;font-family:Arial;'):
            quotes.append(quote.get_text(strip=True))

        return quotes
    
    @staticmethod
    def get_author(table):
        authors = []
        for author in table.find_all('div', style='padding-top:2px;'):
            url = author.find('a').get('href')
            author_name = author.find('a').get_text(strip=True)
            authors.append((author_name, url))

        return authors
    
    @staticmethod
    def get_tags(table):
        tag = table.find('div', style='padding-top:10px;font-size:19px;font-family:Times New Roman;color:#347070;').get_text(strip=True)
        idx = tag.index(' Quote')
        tag = tag[:idx]
        return tag
    
    @staticmethod
    def get_likes(table):
        return None
    
    def scrape_topics(self):
        url = 'http://www.famousquotesandauthors.com/quotes_by_topic.html'
        base_quote_url = 'http://www.famousquotesandauthors.com'
        response = self.get_page(url)
        soup = self.parse_page(response)
        topics = []
        for topic_block in soup.find_all('tr', height='14'):
            topic = topic_block.find('a').get_text(strip=True)
            url = topic_block.find('a').get('href')
            url = urljoin(base_quote_url, url)
            topics.append((topic, url))

        return topics

    def scrape_authors(self):
        url = 'http://www.famousquotesandauthors.com/quotes_by_author.html'
        base_author_url = 'http://www.famousquotesandauthors.com'
        response = self.get_page(url)
        soup = self.parse_page(response)
        authors = []
        for author_block in soup.find_all('tr', height='15'):
            author = author_block.find('a').get_text(strip=True)
            url = author_block.find('a').get('href')
            url = urljoin(base_author_url, url)
            authors.append((author, url))

        return authors
    
if __name__ == "__main__":
    url = 'http://www.famousquotesandauthors.com/topics/worthy_victories_quotes.html'
    scraper = FamousQuotesScraper()
    authors = scraper.scrape_authors()
    print(authors)
