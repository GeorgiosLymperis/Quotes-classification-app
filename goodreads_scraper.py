from base_scraper import BaseScraper
from concurrent.futures import ThreadPoolExecutor, as_completed
from urllib.parse import urljoin
from typing import List

class GoodReadsScraper(BaseScraper):
    def scrape_page(self, url: str, save: bool = False)->List[dict]:
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

        if save == True:
            self._save(quotes, 'quotes_qr.pkl')
        return quotes
    @staticmethod
    def get_quote(block)->str:
        return block.find('div', class_='quoteText').get_text(strip=True)
    
    @staticmethod
    def get_author(block)->str:
        author_tag = block.find('span', class_='authorOrTitle').get_text(strip=True)
        return author_tag if author_tag else 'Unknown'

    @staticmethod
    def get_likes(block)->int:
        likes_box = block.find('div', class_='quoteFooter')
        likes = likes_box.find('a', class_='smallText').get_text(strip=True)
        return likes.split(' ')[0]
    
    @staticmethod
    def get_tags(block)->List[str]:
        tags = []
        tags_block = block.find('div', class_='quoteFooter')
        for tag in tags_block.find_all('a'):
            tags.append(tag.get_text())
        return tags[:-1]
    
    def scrape_many_pages(self, url: str, start_page: int, end_page: int, save: bool = False)->List[dict]:
        quotes = []
        with ThreadPoolExecutor(max_workers=10) as executor:
            futures = []
            for page in self.__generate_pages(url, start_page, end_page):
                future = executor.submit(self.scrape_page, page)
                futures.append(future)

            for future in as_completed(futures):
                quotes.extend(future.result())

        if save == True:
            self._save(quotes, 'quotes_qr.pkl')

        return quotes
    
    def scrape_topics(self, save: bool = False)->List[str]:
        url = 'https://www.goodreads.com/quotes'
        response = self.get_page(url)
        soup = self.parse_page(response)
        topics = []
        for topic in soup.find_all('li', class_='greyText'):
            topic_of_quote = topic.get_text(strip=True).split(' ')[0]
            topic_url = topic.find('a').get('href')
            topic_url = urljoin(url, topic_url)
            topics.append((topic_of_quote, topic_url))

        if save == True:
            self._save(topics, 'topics_qr.pkl')
        return topics
    
    def __generate_pages(self, url, start_page, end_page):
        base_url = urljoin(url, '?page=')
        for i in range(start_page, end_page + 1):
            yield urljoin(base_url, str(i))

if __name__ == '__main__':
    url = 'https://www.goodreads.com/quotes/tag/motivational-quotes'
    scraper = GoodReadsScraper()
    quotes = scraper.scrape_page(url, save=True)
    print(quotes)