import os
import requests
import json
from tqdm import tqdm
import shutil

class ChampionIconDownloader:
    def __init__(self):
        self.base_url = "https://ddragon.leagueoflegends.com"
        self.icons_dir = "assets/champions/icons"
        self.versions_url = f"{self.base_url}/api/versions.json"

    def create_directories(self):
        """Cria os diretórios necessários se não existirem."""
        os.makedirs(self.icons_dir, exist_ok=True)

    def get_latest_version(self):
        """Obtém a versão mais recente do Data Dragon."""
        try:
            response = requests.get(self.versions_url)
            response.raise_for_status()
            versions = response.json()
            return versions[0]
        except requests.RequestException as e:
            print(f"Erro ao obter a versão mais recente: {e}")
            return None

    def get_champions_data(self, version):
        """Obtém os dados de todos os campeões."""
        try:
            url = f"{self.base_url}/cdn/{version}/data/pt_BR/champion.json"
            response = requests.get(url)
            response.raise_for_status()
            return response.json()['data']
        except requests.RequestException as e:
            print(f"Erro ao obter dados dos campeões: {e}")
            return None

    def download_champion_icon(self, champion_name, image_name, version):
        """Baixa o ícone de um campeão específico."""
        try:
            url = f"{self.base_url}/cdn/{version}/img/champion/{image_name}"
            response = requests.get(url, stream=True)
            response.raise_for_status()

            file_path = os.path.join(self.icons_dir, image_name)
            with open(file_path, 'wb') as f:
                response.raw.decode_content = True
                shutil.copyfileobj(response.raw, f)
            return True
        except requests.RequestException as e:
            print(f"Erro ao baixar ícone de {champion_name}: {e}")
            return False

    def run(self):
        """Executa o processo de download dos ícones."""
        print("Iniciando download dos ícones dos campeões...")
        
        # Criar diretórios
        self.create_directories()

        # Obter versão mais recente
        version = self.get_latest_version()
        if not version:
            print("Não foi possível obter a versão mais recente. Abortando.")
            return

        print(f"Usando versão {version} do Data Dragon")

        # Obter dados dos campeões
        champions_data = self.get_champions_data(version)
        if not champions_data:
            print("Não foi possível obter dados dos campeões. Abortando.")
            return

        # Download dos ícones
        total_champions = len(champions_data)
        successful_downloads = 0

        print(f"\nBaixando {total_champions} ícones de campeões...")
        with tqdm(total=total_champions, desc="Progresso", unit="campeão") as pbar:
            for champion_name, champion_data in champions_data.items():
                image_name = champion_data['image']['full']
                if self.download_champion_icon(champion_name, image_name, version):
                    successful_downloads += 1
                pbar.update(1)

        # Relatório final
        print(f"\nDownload concluído!")
        print(f"Total de campeões: {total_champions}")
        print(f"Downloads com sucesso: {successful_downloads}")
        print(f"Falhas: {total_champions - successful_downloads}")
        print(f"\nOs ícones foram salvos em: {os.path.abspath(self.icons_dir)}")

if __name__ == "__main__":
    downloader = ChampionIconDownloader()
    downloader.run()